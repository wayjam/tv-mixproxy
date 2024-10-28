package config

import (
	"io"
	"net/http"

	"github.com/grafov/m3u8"
)

func LoadM3U8Data(uri string) ([]byte, error) {
	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func ParseM3U8Config(r io.Reader) (m3u8.Playlist, m3u8.ListType, error) {
	p, listType, err := m3u8.DecodeFrom(r, true)
	if err != nil {
		return nil, 0, err
	}

	return p, listType, nil
}
