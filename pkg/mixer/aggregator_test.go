package mixer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wayjam/tv-mixproxy/config"
	"github.com/wayjam/tv-mixproxy/pkg/m3u"
)

func TestAggregatorDisabledMerge(t *testing.T) {
	agg := NewAggregator(config.M3UOpt{
		MergeDuplicateChannels: false,
	})

	candidates := []TrackCandidate{
		{Track: m3u.Track{Name: "CCTV-1", URI: "http://stream1.m3u8"}, Priority: 10, OriginalIndex: 0},
		{Track: m3u.Track{Name: "CCTV-1", URI: "http://stream1.m3u8"}, Priority: 10, OriginalIndex: 1}, // duplicate URI
		{Track: m3u.Track{Name: "CCTV-1", URI: "http://stream2.m3u8"}, Priority: 5, OriginalIndex: 2},
	}

	result := agg.Aggregate(candidates)
	assert.Len(t, result, 2)
	assert.Equal(t, "http://stream1.m3u8", result[0].URI)
	assert.Equal(t, "http://stream2.m3u8", result[1].URI)
}

func TestAggregatorMergeRename(t *testing.T) {
	agg := NewAggregator(config.M3UOpt{
		MergeDuplicateChannels: true,
		MergeMode:              "rename",
	})

	candidates := []TrackCandidate{
		{
			Track: m3u.Track{
				Name: "CCTV-1",
				URI:  "http://ipv4.example.com/cctv1.m3u8",
				Tags: []m3u.Tag{{Name: "group-title", Value: "央视"}},
			},
			Priority:      10, // lower priority
			OriginalIndex: 0,
		},
		{
			Track: m3u.Track{
				Name: "CCTV-1",
				URI:  "http://ipv6.example.com/cctv1.m3u8",
				Tags: []m3u.Tag{{Name: "group-title", Value: "央视"}},
			},
			Priority:      100, // higher priority (IPv6 preferred)
			OriginalIndex: 1,
		},
		{
			Track: m3u.Track{
				Name: "湖南卫视",
				URI:  "http://stream.example.com/hunan.m3u8",
				Tags: []m3u.Tag{{Name: "group-title", Value: "卫视"}},
			},
			Priority:      10,
			OriginalIndex: 2,
		},
	}

	result := agg.Aggregate(candidates)
	assert.Len(t, result, 3)

	// CCTV-1 Line 1 should be the higher priority IPv6 stream
	assert.Equal(t, "CCTV-1 (线路1)", result[0].Name)
	assert.Equal(t, "http://ipv6.example.com/cctv1.m3u8", result[0].URI)
	assert.Equal(t, "CCTV-1", result[0].GetTag("tvg-name"))

	// CCTV-1 Line 2 should be the lower priority IPv4 stream
	assert.Equal(t, "CCTV-1 (线路2)", result[1].Name)
	assert.Equal(t, "http://ipv4.example.com/cctv1.m3u8", result[1].URI)
	assert.Equal(t, "CCTV-1", result[1].GetTag("tvg-name"))

	// 湖南卫视 has only 1 line, should not have (线路1) suffix
	assert.Equal(t, "湖南卫视", result[2].Name)
	assert.Equal(t, "http://stream.example.com/hunan.m3u8", result[2].URI)
}

func TestAggregatorMergeDollar(t *testing.T) {
	agg := NewAggregator(config.M3UOpt{
		MergeDuplicateChannels: true,
		MergeMode:              "dollar",
	})

	candidates := []TrackCandidate{
		{
			Track: m3u.Track{
				Name: "CCTV-1",
				URI:  "http://ipv4.example.com/cctv1.m3u8",
				Tags: []m3u.Tag{{Name: "group-title", Value: "央视"}},
			},
			Priority:      10,
			OriginalIndex: 0,
		},
		{
			Track: m3u.Track{
				Name: "CCTV-1",
				URI:  "http://ipv6.example.com/cctv1.m3u8",
				Tags: []m3u.Tag{{Name: "group-title", Value: "央视"}},
			},
			Priority:      100, // Higher priority
			OriginalIndex: 1,
		},
	}

	result := agg.Aggregate(candidates)
	assert.Len(t, result, 1)
	assert.Equal(t, "CCTV-1", result[0].Name)
	assert.Equal(t, "http://ipv6.example.com/cctv1.m3u8$http://ipv4.example.com/cctv1.m3u8", result[0].URI)
}
