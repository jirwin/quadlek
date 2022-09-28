package spotify

import (
	"reflect"
	"testing"
)

func Test_extractSpotifyLink(t *testing.T) {
	type args struct {
		msg string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "single url returned",
			args: args{msg: "https://open.spotify.com/track/0trHOzAhNpGCsGBEu7dOJo?si=b819e61cca094591"},
			want: []string{"0trHOzAhNpGCsGBEu7dOJo"},
		},
		{
			name: "multiple urls returned",
			args: args{msg: "https://open.spotify.com/track/0trHOzAhNpGCsGBEu7dOJo?si=b819e61cca094591 also this https://open.spotify.com/track/7G3lxTsMfSx4yarMkfgnTC?si=29b77185265c4a1e"},
			want: []string{"0trHOzAhNpGCsGBEu7dOJo", "7G3lxTsMfSx4yarMkfgnTC"},
		},
		{
			name: "no urls",
			args: args{msg: "no urls are in this"},
			want: nil,
		},
		{
			name: "old style with URLs",
			args: args{msg: "spotify:track:2037Ob3nwf6lnMCByEoTSB also this https://open.spotify.com/track/7G3lxTsMfSx4yarMkfgnTC?si=29b77185265c4a1e"},
			want: []string{"2037Ob3nwf6lnMCByEoTSB", "7G3lxTsMfSx4yarMkfgnTC"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractSpotifyLink(tt.args.msg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractSpotifyLink() = %v, want %v", got, tt.want)
			}
		})
	}
}
