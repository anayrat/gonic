package funk

import (
	"context"
	"fmt"

	"github.com/machinebox/graphql"
	"github.com/pkg/errors"

	"senan.xyz/g/gonic/model"
)

// TODO: if funk.pub becomes a real thing
// - not store the funk usernames and passwords, use an
//   api key or something similar
// - not sign in for every request
// - probably not use graphql either, audioscrobbler api
//   should be implemented by then

type FunkOptions struct {
	BaseURL  string
	Username string
	Password string
	Track    *model.Track
}

type SignInResponse struct {
	SignIn struct {
		Token string `json:"token"`
	} `json:"signIn"`
}

func Funk(opts FunkOptions) error {
	baseURL := fmt.Sprintf("%s/graphql", opts.BaseURL)
	client := graphql.NewClient(baseURL)
	req := graphql.NewRequest(`
        mutation ($username: String!, $password: String!) {
            signIn(username: $username, password: $password) {
                token
            }
        }
	`)
	req.Var("username", opts.Username)
	req.Var("password", opts.Password)
	resp := &SignInResponse{}
	if err := client.Run(context.Background(), req, resp); err != nil {
		return errors.Wrap(err, "getting token")
	}
	if resp.SignIn.Token == "" {
		return errors.New("token not returned")
	}
	req = graphql.NewRequest(`
		mutation ($artist: String!, $track: String!, $album: String!, albumArt: String!) {
			listen(artist: $artist, track: $track, album: $album, albumArt: $albumArt)
		}
	`)
	req.Var("artist", opts.Track.TagTrackArtist)
	req.Var("track", opts.Track.TagTitle)
	req.Var("album", opts.Track.Album.TagTitle)
	req.Var("albumArt", fmt.Sprintf("https://gonic.home.senan.xyz/rest/getCoverArt.view?id=%d", opts.Track.Album.ID))
	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", resp.SignIn.Token))
	if err := client.Run(context.Background(), req, nil); err != nil {
		return errors.Wrap(err, "sending listen")
	}
	return nil
}
