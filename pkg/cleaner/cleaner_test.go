package cleaner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wayjam/tv-mixproxy/config"
	"github.com/wayjam/tv-mixproxy/pkg/m3u"
)

func TestCleanName(t *testing.T) {
	engine := NewEngine(
		config.CleanerOpt{Enable: true},
		config.EPGMappingOpt{Enable: false},
	)

	tests := []struct {
		input    string
		expected string
	}{
		{"CCTV-1 高清 [1080P]", "CCTV-1"},
		{"[IPv6] CCTV1 综合", "CCTV-1"},
		{"CCTV5+ 4K 60fps", "CCTV-5+"},
		{"湖南卫视 HD [超清]", "湖南卫视"},
		{"浙江卫视高清", "浙江卫视"},
		{"北京卫视", "北京卫视"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := engine.CleanName(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestProcessTrack(t *testing.T) {
	engine := NewEngine(
		config.CleanerOpt{Enable: true},
		config.EPGMappingOpt{
			Enable:          true,
			LogoURLTemplate: "https://example.com/logo/{name}.png",
			Mappings: map[string]string{
				"CCTV-1": "cctv1.epg",
			},
		},
	)

	track := &m3u.Track{
		Name: "CCTV1 高清 [IPv6]",
		URI:  "http://example.com/cctv1.m3u8",
	}

	engine.ProcessTrack(track)

	assert.Equal(t, "CCTV-1", track.Name)
	assert.Equal(t, "CCTV-1", track.GetTag("tvg-name"))
	assert.Equal(t, "cctv1.epg", track.GetTag("tvg-id"))
	assert.Equal(t, "https://example.com/logo/CCTV-1.png", track.GetTag("tvg-logo"))
}

func TestCustomAliases(t *testing.T) {
	engine := NewEngine(
		config.CleanerOpt{
			Enable: true,
			Aliases: []config.AliasRule{
				{
					Canonical: "翡翠台",
					Matches:   []string{"Jade HD", "香港翡翠台"},
				},
			},
		},
		config.EPGMappingOpt{Enable: false},
	)

	assert.Equal(t, "翡翠台", engine.CleanName("Jade HD [1080P]"))
	assert.Equal(t, "翡翠台", engine.CleanName("香港翡翠台"))
}

func TestCleanerDisabled(t *testing.T) {
	engine := NewEngine(
		config.CleanerOpt{Enable: false},
		config.EPGMappingOpt{Enable: false},
	)

	name := "CCTV1 高清 [1080P]"
	assert.Equal(t, name, engine.CleanName(name))

	track := &m3u.Track{Name: name, URI: "http://example.com/stream.m3u8"}
	engine.ProcessTrack(track)
	assert.Equal(t, name, track.Name)
	assert.Empty(t, track.GetTag("tvg-logo"))
}
