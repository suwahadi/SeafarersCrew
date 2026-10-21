package video

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseRecognizesYouTubeAndVimeoLinks(t *testing.T) {
	cases := []struct {
		in   string
		want Ref
		ok   bool
	}{
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", Ref{YouTube, "dQw4w9WgXcQ"}, true},
		{"https://youtu.be/dQw4w9WgXcQ?t=10", Ref{YouTube, "dQw4w9WgXcQ"}, true},
		{"https://www.youtube.com/embed/dQw4w9WgXcQ", Ref{YouTube, "dQw4w9WgXcQ"}, true},
		{"https://youtube.com/shorts/dQw4w9WgXcQ", Ref{YouTube, "dQw4w9WgXcQ"}, true},
		{"https://vimeo.com/76979871", Ref{Vimeo, "76979871"}, true},
		{"https://vimeo.com/76979871/abcdef1234", Ref{Vimeo, "76979871"}, true},
		{"https://player.vimeo.com/video/76979871", Ref{Vimeo, "76979871"}, true},
		{"https://www.youtube.com/watch?v=short", Ref{}, false},
		{"https://example.com/watch?v=dQw4w9WgXcQ", Ref{}, false},
		{"https://evil.com/youtube.com/watch?v=dQw4w9WgXcQ", Ref{}, false},
		{"javascript:alert(1)", Ref{}, false},
		{"https://vimeo.com/channels/staffpicks", Ref{}, false},
		{"", Ref{}, false},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, ok := Parse(c.in)
			assert.Equal(t, c.ok, ok)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestRefBuildsEmbedAndThumbnailURLs(t *testing.T) {
	yt := Ref{YouTube, "dQw4w9WgXcQ"}
	assert.Equal(t, "https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ", yt.EmbedURL())
	assert.Equal(t, "https://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg", yt.ThumbnailURL())
	vm := Ref{Vimeo, "76979871"}
	assert.Equal(t, "https://player.vimeo.com/video/76979871", vm.EmbedURL())
	assert.Empty(t, vm.ThumbnailURL())
}
