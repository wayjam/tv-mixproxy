package config

import (
	"io"

	"github.com/wayjam/tv-mixproxy/pkg/epg"
)

func ParseEPGConfig(r io.Reader) (*epg.EPG, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return epg.Unmarshal(data)
}
