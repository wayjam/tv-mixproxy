package mixer

import (
	"github.com/wayjam/tv-mixproxy/config"
	"github.com/wayjam/tv-mixproxy/pkg/txt"
)

// MixTXT 混合直播源并转换为标准 TXT/DIYP 格式
func MixTXT(cfg *config.Config, sourcer Sourcer, opts ...MixOptions) (*txt.Playlist, error) {
	m3uPl, err := MixM3UMediaPlayList(cfg, sourcer, opts...)
	if err != nil {
		return nil, err
	}
	if m3uPl == nil {
		return &txt.Playlist{}, nil
	}
	return txt.FromM3U(m3uPl), nil
}
