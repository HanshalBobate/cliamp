package ytplaylist

import (
	"cliamp/playlist"
	"cliamp/resolve"
	"fmt"
)

// Provider serves a single YouTube playlist as a provider, scraping songs automatically.
type Provider struct {
	url string
}

// New creates a new YouTube playlist provider.
func New(url string) *Provider {
	return &Provider{url: url}
}

func (p *Provider) Name() string { return "YouTube Playlist" }

// Playlists returns a single entry representing this playlist.
func (p *Provider) Playlists() ([]playlist.PlaylistInfo, error) {
	return []playlist.PlaylistInfo{
		{ID: "0", Name: "Scraped Playlist Tracks"},
	}, nil
}

// Tracks fetches all tracks from the YouTube playlist using yt-dlp via resolve.
func (p *Provider) Tracks(id string) ([]playlist.Track, error) {
	if id != "0" {
		return nil, fmt.Errorf("ytplaylist: unknown playlist ID %q", id)
	}
	tracks, err := resolve.ResolveYTDLBatch(p.url, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("scraping playlist: %w", err)
	}
	return tracks, nil
}
