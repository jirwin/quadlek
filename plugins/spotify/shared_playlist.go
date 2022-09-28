package spotify

import (
	"context"
	"net/url"
	"os"
	"regexp"
	"strings"

	v1 "github.com/jirwin/quadlek/pb/quadlek/plugins/spotify/v1"
	"mvdan.cc/xurls/v2"

	"github.com/jirwin/quadlek/quadlek"
	"github.com/zmb3/spotify"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const (
	sharedPlaylist     = "2oeMx9gAl3fx2qThl11jt1"
	sharedPlaylistUser = "U0RL53ETW"
)

var spotifyTrackRegex = regexp.MustCompile(`spotify:track:(\w+)\b`)

func getSharedPlaylist() string {
	playlist := os.Getenv("SPOTIFY_SHARED_PLAYLIST")
	if playlist != "" {
		return playlist
	}

	return sharedPlaylist
}

func getSharedPlaylistUser() string {
	playlistUser := os.Getenv("SPOTIFY_SHARED_PLAYLIST_USER")
	if playlistUser != "" {
		return playlistUser
	}

	return sharedPlaylistUser
}

func extractSpotifyLink(msg string) []string {
	var ret []string
	matches := spotifyTrackRegex.FindAllStringSubmatch(msg, -1)
	for _, m := range matches {
		ret = append(ret, m[1])
	}
	rxStrict := xurls.Strict()
	urls := rxStrict.FindAllString(msg, -1)
	for _, u := range urls {
		parsed, err := url.Parse(u)
		if err != nil {
			continue
		}

		if parsed.Scheme != "https" || parsed.Hostname() != "open.spotify.com" {
			continue
		}

		if strings.HasPrefix(parsed.Path, "/track/") {
			ret = append(ret, strings.TrimPrefix(parsed.Path, "/track/"))
		}
	}

	return ret
}

func saveSongsHook(ctx context.Context, hookChannel <-chan *quadlek.HookMsg) {
	for {
		select {
		case hookMsg := <-hookChannel:
			var tracks []spotify.ID
			for _, match := range extractSpotifyLink(hookMsg.Msg.Text) {
				tracks = append(tracks, spotify.ID(match))
			}

			err := hookMsg.Store.Get("authtoken-"+getSharedPlaylistUser(), func(val []byte) error {
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

				snapshotId, err := client.AddTracksToPlaylist(spotify.ID(getSharedPlaylist()), tracks...)
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
