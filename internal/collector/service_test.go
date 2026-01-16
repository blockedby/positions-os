package collector

import (
	"testing"
)

func TestBuildSourceURL(t *testing.T) {
	tests := []struct {
		name       string
		channelURL string
		messageID  int
		want       string
	}{
		{
			name:       "channel with @ prefix",
			channelURL: "@stablegram",
			messageID:  1244,
			want:       "https://t.me/stablegram/1244",
		},
		{
			name:       "channel without @ prefix",
			channelURL: "golang_jobs",
			messageID:  100,
			want:       "https://t.me/golang_jobs/100",
		},
		{
			name:       "channel with https prefix",
			channelURL: "https://t.me/remote_it",
			messageID:  500,
			want:       "https://t.me/remote_it/500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildSourceURL(tt.channelURL, tt.messageID)
			if got != tt.want {
				t.Errorf("buildSourceURL() = %s, want %s", got, tt.want)
			}
		})
	}
}
