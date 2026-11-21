package mixer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wayjam/tv-mixproxy/config"
)

func TestQueryDIYPEPG(t *testing.T) {
	epgData := `<?xml version="1.0" encoding="UTF-8"?>
<tv>
    <channel id="CCTV1">
        <display-name>CCTV-1 综合</display-name>
    </channel>
    <channel id="HUNAN">
        <display-name>湖南卫视</display-name>
    </channel>
    <programme channel="CCTV1" start="20261121083000 +0800" stop="20261121090000 +0800">
        <title>早间新闻</title>
    </programme>
    <programme channel="CCTV1" start="20261121190000 +0800" stop="20261121193000 +0800">
        <title>新闻联播</title>
    </programme>
    <programme channel="CCTV1" start="20261122190000 +0800" stop="20261122193000 +0800">
        <title>次日新闻联播</title>
    </programme>
    <programme channel="HUNAN" start="20261121200000 +0800" stop="20261121210000 +0800">
        <title>快乐大本营</title>
    </programme>
</tv>`

	mockSourcer := &MockSourcer{
		sources: map[string]*Source{
			"epg_src": {
				config: config.Source{Name: "epg_src", Type: config.SourceTypeEPG},
				data:   []byte(epgData),
			},
		},
	}

	cfg := &config.Config{
		EPGOpt: config.EPGOpt{
			Disable: false,
			Filters: []config.ArrayMixOpt{
				{MixOpt: config.MixOpt{SourceName: "epg_src"}},
			},
		},
	}

	// 1. Fuzzy match channel name "CCTV1" and date "2026-11-21"
	resp1, err := QueryDIYPEPG(cfg, mockSourcer, "CCTV1", "2026-11-21")
	assert.NoError(t, err)
	assert.NotNil(t, resp1)
	assert.Equal(t, "CCTV1", resp1.ChannelName)
	assert.Equal(t, "2026-11-21", resp1.Date)
	assert.Len(t, resp1.EPGData, 2)
	assert.Equal(t, "08:30", resp1.EPGData[0].Start)
	assert.Equal(t, "09:00", resp1.EPGData[0].End)
	assert.Equal(t, "早间新闻", resp1.EPGData[0].Title)
	assert.Equal(t, "19:00", resp1.EPGData[1].Start)
	assert.Equal(t, "19:30", resp1.EPGData[1].End)
	assert.Equal(t, "新闻联播", resp1.EPGData[1].Title)

	// 2. Query next day date "2026-11-22"
	resp2, err := QueryDIYPEPG(cfg, mockSourcer, "CCTV-1 综合", "2026-11-22")
	assert.NoError(t, err)
	assert.Len(t, resp2.EPGData, 1)
	assert.Equal(t, "次日新闻联播", resp2.EPGData[0].Title)

	// 3. Query non-existent channel
	resp3, err := QueryDIYPEPG(cfg, mockSourcer, "NonExistent", "2026-11-21")
	assert.NoError(t, err)
	assert.Empty(t, resp3.EPGData)
}
