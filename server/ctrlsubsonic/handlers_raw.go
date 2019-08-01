package ctrlsubsonic

import (
	"net/http"
	"path"
	"time"

	"github.com/jinzhu/gorm"

	"senan.xyz/g/gonic/model"
	"senan.xyz/g/gonic/server/ctrlsubsonic/spec"
	"senan.xyz/g/gonic/server/key"
	"senan.xyz/g/gonic/server/parsing"
)

// "raw" handlers are ones that don't always return a spec response.
// it could be a file, stream, etc. so you must either
//   a) write to response writer
//   b) return a non-nil spec.Response
//  _but not both_

func (c *Controller) ServeGetCoverArt(w http.ResponseWriter, r *http.Request) *spec.Response {
	id, err := parsing.GetIntParam(r, "id")
	if err != nil {
		return spec.NewError(10, "please provide an `id` parameter")
	}
	folder := &model.Album{}
	err = c.DB.
		Select("id, left_path, right_path, cover").
		First(folder, id).
		Error
	if gorm.IsRecordNotFoundError(err) {
		return spec.NewError(10, "could not find a cover with that id")
	}
	if folder.Cover == "" {
		return spec.NewError(10, "no cover found for that folder")
	}
	absPath := path.Join(
		c.MusicPath,
		folder.LeftPath,
		folder.RightPath,
		folder.Cover,
	)
	http.ServeFile(w, r, absPath)
	return nil
}

func (c *Controller) ServeStream(w http.ResponseWriter, r *http.Request) *spec.Response {
	id, err := parsing.GetIntParam(r, "id")
	if err != nil {
		return spec.NewError(10, "please provide an `id` parameter")
	}
	track := &model.Track{}
	err = c.DB.
		Preload("Album").
		First(track, id).
		Error
	if gorm.IsRecordNotFoundError(err) {
		return spec.NewError(70, "media with id `%d` was not found", id)
	}
	absPath := path.Join(
		c.MusicPath,
		track.Album.LeftPath,
		track.Album.RightPath,
		track.Filename,
	)
	http.ServeFile(w, r, absPath)
	//
	// after we've served the file, mark the album as played
	user := r.Context().Value(key.User).(*model.User)
	play := model.Play{
		AlbumID: track.Album.ID,
		UserID:  user.ID,
	}
	c.DB.
		Where(play).
		First(&play)
	play.Time = time.Now() // for getAlbumList?type=recent
	play.Count++           // for getAlbumList?type=frequent
	c.DB.Save(&play)
	return nil
}
