package ctrlsubsonic

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/jinzhu/gorm"

	"github.com/sentriz/gonic/model"
	"github.com/sentriz/gonic/server/ctrlsubsonic/spec"
	"github.com/sentriz/gonic/server/key"
	"github.com/sentriz/gonic/server/parsing"
)

// the subsonic spec metions "artist" a lot when talking about the
// browse by folder endpoints. but since we're not browsing by tag
// we can't access artists. so instead we'll consider the artist of
// an track to be the it's respective folder that comes directly
// under the root directory

func (c *Controller) ServeGetIndexes(r *http.Request) *spec.Response {
	var folders []*model.Album
	c.DB.
		Select("*, count(sub.id) as child_count").
		Joins(`
            LEFT JOIN albums sub
		    ON albums.id = sub.parent_id
		`).
		Where("albums.parent_id = 1").
		Group("albums.id").
		Find(&folders)
	// [a-z#] -> 27
	indexMap := make(map[string]*spec.Index, 27)
	resp := make([]*spec.Index, 0, 27)
	for _, folder := range folders {
		i := lowerUDecOrHash(folder.IndexRightPath())
		index, ok := indexMap[i]
		if !ok {
			index = &spec.Index{
				Name:    i,
				Artists: []*spec.Artist{},
			}
			indexMap[i] = index
			resp = append(resp, index)
		}
		index.Artists = append(index.Artists,
			spec.NewArtistByFolder(folder))
	}
	sort.Slice(resp, func(i, j int) bool {
		return resp[i].Name < resp[j].Name
	})
	sub := spec.NewResponse()
	sub.Indexes = &spec.Indexes{
		LastModified: 0,
		Index:        resp,
	}
	return sub
}

func (c *Controller) ServeGetMusicDirectory(r *http.Request) *spec.Response {
	id, err := parsing.GetIntParam(r, "id")
	if err != nil {
		return spec.NewError(10, "please provide an `id` parameter")
	}
	childrenObj := []*spec.TrackChild{}
	folder := &model.Album{}
	c.DB.First(folder, id)
	//
	// start looking for child childFolders in the current dir
	var childFolders []*model.Album
	c.DB.
		Where("parent_id = ?", id).
		Find(&childFolders)
	for _, c := range childFolders {
		childrenObj = append(childrenObj, spec.NewTCAlbumByFolder(c))
	}
	//
	// start looking for child childTracks in the current dir
	var childTracks []*model.Track
	c.DB.
		Where("album_id = ?", id).
		Preload("Album").
		Order("filename").
		Find(&childTracks)
	for _, c := range childTracks {
		toAppend := spec.NewTCTrackByFolder(c, folder)
		if parsing.GetStrParam(r, "c") == "Jamstash" {
			// jamstash thinks it can't play flacs
			toAppend.ContentType = "audio/mpeg"
			toAppend.Suffix = "mp3"
		}
		childrenObj = append(childrenObj, toAppend)
	}
	//
	// respond section
	sub := spec.NewResponse()
	sub.Directory = spec.NewDirectoryByFolder(folder, childrenObj)
	return sub
}

// changes to this function should be reflected in in _by_tags.go's
// getAlbumListTwo() function
func (c *Controller) ServeGetAlbumList(r *http.Request) *spec.Response {
	listType := parsing.GetStrParam(r, "type")
	if listType == "" {
		return spec.NewError(10, "please provide a `type` parameter")
	}
	q := c.DB.DB
	switch listType {
	case "alphabeticalByArtist":
		q = q.Joins(`
			JOIN albums AS parent_albums
			ON albums.parent_id = parent_albums.id`)
		q = q.Order("parent_albums.right_path")
	case "alphabeticalByName":
		q = q.Order("right_path")
	case "frequent":
		user := r.Context().Value(key.User).(*model.User)
		q = q.Joins(`
			JOIN plays
			ON albums.id = plays.album_id AND plays.user_id = ?`,
			user.ID)
		q = q.Order("plays.count DESC")
	case "newest":
		q = q.Order("modified_at DESC")
	case "random":
		q = q.Order(gorm.Expr("random()"))
	case "recent":
		user := r.Context().Value(key.User).(*model.User)
		q = q.Joins(`
			JOIN plays
			ON albums.id = plays.album_id AND plays.user_id = ?`,
			user.ID)
		q = q.Order("plays.time DESC")
	default:
		return spec.NewError(10, "unknown value `%s` for parameter 'type'", listType)
	}
	var folders []*model.Album
	q.
		Where("albums.tag_artist_id IS NOT NULL").
		Offset(parsing.GetIntParamOr(r, "offset", 0)).
		Limit(parsing.GetIntParamOr(r, "size", 10)).
		Preload("Parent").
		Find(&folders)
	sub := spec.NewResponse()
	sub.Albums = &spec.Albums{
		List: make([]*spec.Album, len(folders)),
	}
	for i, folder := range folders {
		sub.Albums.List[i] = spec.NewAlbumByFolder(folder)
	}
	return sub
}

func (c *Controller) ServeSearchTwo(r *http.Request) *spec.Response {
	query := parsing.GetStrParam(r, "query")
	if query == "" {
		return spec.NewError(10, "please provide a `query` parameter")
	}
	query = fmt.Sprintf("%%%s%%", strings.TrimSuffix(query, "*"))
	results := &spec.SearchResultTwo{}
	//
	// search "artists"
	var artists []*model.Album
	c.DB.
		Where(`
            parent_id = 1
            AND (right_path LIKE ? OR
                 right_path_u_dec LIKE ?)
		`, query, query).
		Offset(parsing.GetIntParamOr(r, "artistOffset", 0)).
		Limit(parsing.GetIntParamOr(r, "artistCount", 20)).
		Find(&artists)
	for _, a := range artists {
		results.Artists = append(results.Artists,
			spec.NewDirectoryByFolder(a, nil))
	}
	//
	// search "albums"
	var albums []*model.Album
	c.DB.
		Where(`
            tag_artist_id IS NOT NULL
            AND (right_path LIKE ? OR
                 right_path_u_dec LIKE ?)
		`, query, query).
		Offset(parsing.GetIntParamOr(r, "albumOffset", 0)).
		Limit(parsing.GetIntParamOr(r, "albumCount", 20)).
		Find(&albums)
	for _, a := range albums {
		results.Albums = append(results.Albums, spec.NewTCAlbumByFolder(a))
	}
	//
	// search tracks
	var tracks []*model.Track
	c.DB.
		Preload("Album").
		Where(`
            filename LIKE ? OR
            filename_u_dec LIKE ?
		`, query, query).
		Offset(parsing.GetIntParamOr(r, "songOffset", 0)).
		Limit(parsing.GetIntParamOr(r, "songCount", 20)).
		Find(&tracks)
	for _, t := range tracks {
		results.Tracks = append(results.Tracks,
			spec.NewTCTrackByFolder(t, t.Album))
	}
	//
	sub := spec.NewResponse()
	sub.SearchResultTwo = results
	return sub
}
