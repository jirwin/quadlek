package spotify

import (
	"context"
	v1 "github.com/jirwin/quadlek/pb/quadlek/plugins/spotify/v1"

	"go.uber.org/zap"

	"regexp"

	"github.com/jirwin/quadlek/quadlek"
	"github.com/zmb3/spotify"
	"google.golang.org/protobuf/proto"
)

// FIXME: These shouldn't have to be hardcoded
const (
	sharedPlaylist     = "2oeMx9gAl3fx2qThl11jt1"
	sharedPlaylistUser = "U0RL53ETW"
)

var spotifyTrackRegex = regexp.MustCompile(`spotify:track:(\w+)\b`)

func saveSongsHook(ctx context.Context, hookChannel <-chan *quadlek.HookMsg) {
	for {
		select {
		case hookMsg := <-hookChannel:
			matches := spotifyTrackRegex.FindAllStringSubmatch(hookMsg.Msg.Text, -1)
			if matches == nil {
				continue
			}

			tracks := []spotify.ID{}
			for _, match := range matches {
				tracks = append(tracks, spotify.ID(match[1]))
			}

			err := hookMsg.Store.Get("authtoken-"+sharedPlaylistUser, func(val []byte) error {
				authToken := &v1.AuthToken{}
				err := proto.Unmarshal(val, authToken)
				if err != nil {
					return err
				}

				if authToken.Token == nil {
					zap.L().Error("There wasn't a token for the shared playlist user")
					return nil
				}
				client, needsReauth := getSpotifyClient(authToken)
				if needsReauth {
					zap.L().Info("detected a song, but need to reauth before it can be added.")
				}

				snapshotId, err := client.AddTracksToPlaylist(sharedPlaylist, tracks...)
				if err != nil {
					zap.L().Error("error adding tracks to shared playlist", zap.Error(err))
					return err
				}
				zap.L().Info("Spotify snapshot id", zap.String("snapshotId", snapshotId))

				return nil
			})
			if err != nil {
				continue
			}

		case <-ctx.Done():
			zap.L().Info("Exiting save song hook")
			return
		}
	}
}
