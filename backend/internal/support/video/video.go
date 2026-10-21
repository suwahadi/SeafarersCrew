// Package video parses YouTube and Vimeo links.
package video

import (
	"net/url"
	"regexp"
	"strings"
)

// Providers supported by the video block.
const (
	YouTube = "youtube"
	Vimeo   = "vimeo"
)

// Ref identifies one hosted video.
type Ref struct {
	Provider string
	ID       string
}

var (
	youtubeID = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
	vimeoID   = regexp.MustCompile(`^[0-9]{1,12}$`)
)

var youtubeHosts = map[string]bool{
	"youtube.com": true, "www.youtube.com": true, "m.youtube.com": true,
	"youtube-nocookie.com": true, "www.youtube-nocookie.com": true,
}

var vimeoHosts = map[string]bool{"vimeo.com": true, "www.vimeo.com": true, "player.vimeo.com": true}

// Parse extracts the provider and video ID from a YouTube or Vimeo URL.
func Parse(raw string) (Ref, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return Ref{}, false
	}
	host := strings.ToLower(u.Hostname())
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	switch {
	case host == "youtu.be":
		return youtubeRef(parts[0])
	case youtubeHosts[host]:
		return parseYouTube(u, parts)
	case vimeoHosts[host]:
		return parseVimeo(host, parts)
	}
	return Ref{}, false
}

func parseYouTube(u *url.URL, parts []string) (Ref, bool) {
	if len(parts) == 1 && parts[0] == "watch" {
		return youtubeRef(u.Query().Get("v"))
	}
	if len(parts) == 2 && (parts[0] == "embed" || parts[0] == "shorts" || parts[0] == "live" || parts[0] == "v") {
		return youtubeRef(parts[1])
	}
	return Ref{}, false
}

func parseVimeo(host string, parts []string) (Ref, bool) {
	if host == "player.vimeo.com" {
		if len(parts) == 2 && parts[0] == "video" {
			return vimeoRef(parts[1])
		}
		return Ref{}, false
	}
	if len(parts) >= 1 && len(parts) <= 2 {
		return vimeoRef(parts[0])
	}
	return Ref{}, false
}

func youtubeRef(id string) (Ref, bool) {
	if !youtubeID.MatchString(id) {
		return Ref{}, false
	}
	return Ref{Provider: YouTube, ID: id}, true
}

func vimeoRef(id string) (Ref, bool) {
	if !vimeoID.MatchString(id) {
		return Ref{}, false
	}
	return Ref{Provider: Vimeo, ID: id}, true
}

// EmbedURL returns the privacy-friendly iframe URL.
func (r Ref) EmbedURL() string {
	if r.Provider == YouTube {
		return "https://www.youtube-nocookie.com/embed/" + r.ID
	}
	return "https://player.vimeo.com/video/" + r.ID
}

// ThumbnailURL returns a poster image URL, or "" when the provider offers none without an API call.
func (r Ref) ThumbnailURL() string {
	if r.Provider == YouTube {
		return "https://i.ytimg.com/vi/" + r.ID + "/hqdefault.jpg"
	}
	return ""
}
