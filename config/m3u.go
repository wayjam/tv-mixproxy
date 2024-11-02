package config

import (
	"io"

	"github.com/wayjam/tv-mixproxy/pkg/m3u"
)

func ParseM3U8Config(r io.Reader) (m3u.Playlist, error) {
	var playlist m3u.Playlist
	data, err := io.ReadAll(r)
	if err != nil {
		return playlist, err
	}

	err = m3u.Unmarshal(data, &playlist)
	if err != nil {
		return playlist, err
	}

	return playlist, nil
}
