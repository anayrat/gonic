package spec

import "github.com/sentriz/gonic/model"

func NewPlaylist(p *model.Playlist) *Playlist {
	return &Playlist{
		ID:       p.ID,
		Name:     p.Name,
		Comment:  p.Comment,
		Duration: "1",
		Public:   true,
	}
}
