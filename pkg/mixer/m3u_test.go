package mixer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wayjam/tv-mixproxy/config"
)

func TestMixM3UMediaPlayList(t *testing.T) {
	sourceData1 := `#EXTM3U
#EXTINF:-1 tvg-id="cctv1" tvg-name="CCTV1" group-title="央视",CCTV-1 综合
http://example.com/cctv1.m3u8
#EXTINF:-1 tvg-id="cctv2" tvg-name="CCTV2" group-title="央视",CCTV-2 财经
http://example.com/cctv2.m3u8
`
	sourceData2 := `#EXTM3U
#EXTINF:-1 tvg-id="hunan" tvg-name="湖南卫视" group-title="卫视",湖南卫视
http://example.com/hunan.m3u8
#EXTINF:-1 tvg-id="zhejiang" tvg-name="浙江卫视" group-title="卫视",浙江卫视
http://example.com/zhejiang.m3u8
`

	mockSourcer := &MockSourcer{
		sources: map[string]*Source{
			"src1": {
				config: config.Source{Name: "src1", Type: config.SourceTypeM3U},
				data:   []byte(sourceData1),
			},
			"src2": {
				config: config.Source{Name: "src2", Type: config.SourceTypeM3U},
				data:   []byte(sourceData2),
			},
		},
	}

	t.Run("Filter by group-title", func(t *testing.T) {
		cfg := &config.Config{
			ServerPort:  8080,
			ExternalURL: "http://proxy.test",
			M3UOpt: config.M3UOpt{
				MediaPlaylistFilters: []config.ArrayMixOpt{
					{
						MixOpt:   config.MixOpt{SourceName: "src1"},
						FilterBy: "group-title",
						Include:  "央视",
					},
					{
						MixOpt:   config.MixOpt{SourceName: "src2"},
						FilterBy: "tvg-name",
						Include:  "湖南.*",
					},
				},
			},
			EPGOpt: config.EPGOpt{Disable: false},
		}

		result, err := MixM3UMediaPlayList(cfg, mockSourcer)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// 2 from src1 (both 央视) + 1 from src2 (湖南卫视)
		assert.Len(t, result.Tracks, 3)
		assert.Equal(t, "CCTV-1 综合", result.Tracks[0].Name)
		assert.Equal(t, "CCTV-2 财经", result.Tracks[1].Name)
		assert.Equal(t, "湖南卫视", result.Tracks[2].Name)

		// Verify auto-linked EPG
		assert.Equal(t, "http://proxy.test/v1/epg.xml", result.GetTag("x-tvg-url"))
	})

	t.Run("Cleaner and EPGMapping enabled", func(t *testing.T) {
		cfg := &config.Config{
			ServerPort:  8080,
			ExternalURL: "http://proxy.test",
			M3UOpt: config.M3UOpt{
				DefaultGroup: "央视频道",
				EPGURLs:      []string{"{external_url}/v1/epg.xml", "http://backup.epg/e.xml"},
				MediaPlaylistFilters: []config.ArrayMixOpt{
					{
						MixOpt:   config.MixOpt{SourceName: "src1"},
						FilterBy: "name",
						Include:  "CCTV-1.*",
					},
				},
			},
			CleanerOpt: config.CleanerOpt{
				Enable: true,
			},
			EPGMappingOpt: config.EPGMappingOpt{
				Enable:          true,
				LogoURLTemplate: "http://logo.test/{name}.png",
				Mappings: map[string]string{
					"CCTV-1": "cctv1.epg.id",
				},
			},
		}

		result, err := MixM3UMediaPlayList(cfg, mockSourcer)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Tracks, 1)

		track := result.Tracks[0]
		// Name cleaned from "CCTV-1 综合" to "CCTV-1"
		assert.Equal(t, "CCTV-1", track.Name)
		// tvg-id mapped to "cctv1.epg.id"
		assert.Equal(t, "cctv1.epg.id", track.GetTag("tvg-id"))
		// tvg-logo mapped to "http://logo.test/CCTV-1.png"
		assert.Equal(t, "http://logo.test/CCTV-1.png", track.GetTag("tvg-logo"))
		// group-title preserved from original tag
		assert.Equal(t, "央视", track.GetTag("group-title"))

		// Custom multi-EPG URLs
		assert.Equal(t, "http://proxy.test/v1/epg.xml,http://backup.epg/e.xml", result.GetTag("x-tvg-url"))
	})
}
