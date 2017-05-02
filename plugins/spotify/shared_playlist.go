package spotify

import (
	"context"

	"regexp"

	log "github.com/Sirupsen/logrus"
	"github.com/golang/protobuf/proto"
	"github.com/jirwin/quadlek/quadlek"
	"github.com/zmb3/spotify"
)

// FIXME: These shouldn't have to be hardcoded
const (
	sharedPlaylist         = "2oeMx9gAl3fx2qThl11jt1"
	sharedPlaylistUser     = "U0RL53ETW"
	sharedPlaylistUsername = "jirwin304"
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
				authToken := &AuthToken{}
				err := proto.Unmarshal(val, authToken)
				if err != nil {
					return err
				}

				if authToken.Token == nil {
					log.Error("There wasn't a token for the shared playlist user")
					return nil
				}
				client, needsReauth := getSpotifyClient(authToken)
				if needsReauth {
					log.Info("detected a song, but need to reauth before it can be added.")
				}

				snapshotId, err := client.AddTracksToPlaylist(sharedPlaylistUsername, sharedPlaylist, tracks...)
				if err != nil {
					log.WithFields(log.Fields{
						"err":    err,
						"tracks": tracks,
					}).Error("error adding tracks to shared playlist")
					return err
				}
				log.Info("Spotify snapshot id: ", snapshotId)

				return nil
			})
			if err != nil {
				continue
			}

		case <-ctx.Done():
			log.Info("Exiting save song hook")
			return
		}
	}
}
