package mixer

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wayjam/tv-mixproxy/config"
	"github.com/wayjam/tv-mixproxy/pkg/epg"
)

// MockEPGSourcer 是一个用于测试的 Sourcer 实现
type MockEPGSourcer struct {
	sources map[string]*Source
}

func (m *MockEPGSourcer) GetSource(name string) (*Source, error) {
	source, ok := m.sources[name]
	if !ok {
		return nil, fmt.Errorf("source not found: %s", name)
	}
	return source, nil
}

func TestMixEPG(t *testing.T) {
	// 创建测试用的EPG数据
	epgData1 := `<?xml version="1.0" encoding="UTF-8"?>
<tv>
    <channel id="channel1">
        <display-name>Channel 1</display-name>
    </channel>
    <channel id="channel2">
        <display-name>Channel 2</display-name>
    </channel>
    <programme channel="channel1" start="20240101000000" stop="20240101010000">
        <title>Show A</title>
    </programme>
    <programme channel="channel2" start="20240101000000" stop="20240101010000">
        <title>Show B</title>
    </programme>
</tv>`

	epgData2 := `<?xml version="1.0" encoding="UTF-8"?>
<tv>
    <channel id="channel3">
        <display-name>Channel 3</display-name>
    </channel>
    <programme channel="channel3" start="20240101000000" stop="20240101010000">
        <title>Special Show</title>
    </programme>
</tv>`

	tests := []struct {
		name           string
		config         *config.Config
		expectedCount  int
		expectedError  bool
		checkFunction  func(*epg.EPG) bool
		expectedSource string
	}{
		{
			name: "Filter by channel ID",
			config: &config.Config{
				EPGOpt: config.EPGOpt{
					Disable: false,
					Filters: []config.ArrayMixOpt{
						{
							MixOpt: config.MixOpt{
								SourceName: "source1",
							},
							FilterBy: string(config.EPGFilterTypeChannelID),
							Include:  "channel1",
						},
					},
				},
			},
			expectedCount: 1,
			checkFunction: func(epg *epg.EPG) bool {
				return len(epg.Channel) == 1 && epg.Channel[0].ID == "channel1" &&
					len(epg.Programme) == 1 && epg.Programme[0].Channel == "channel1"
			},
		},
		{
			name: "Filter by program title",
			config: &config.Config{
				EPGOpt: config.EPGOpt{
					Disable: false,
					Filters: []config.ArrayMixOpt{
						{
							MixOpt: config.MixOpt{
								SourceName: "source2",
							},
							FilterBy: string(config.EPGFilterTypeProgramTitle),
							Include:  "Special",
						},
					},
				},
			},
			expectedCount: 1,
			checkFunction: func(epg *epg.EPG) bool {
				return len(epg.Channel) == 1 && epg.Channel[0].ID == "channel3" &&
					len(epg.Programme) == 1 && epg.Programme[0].Title.Text == "Special Show"
			},
		},
		{
			name: "EPG disabled",
			config: &config.Config{
				EPGOpt: config.EPGOpt{
					Disable: true,
				},
			},
			expectedCount: 0,
			checkFunction: func(epg *epg.EPG) bool {
				return len(epg.Channel) == 0 && len(epg.Programme) == 0
			},
		},
		{
			name: "Invalid source",
			config: &config.Config{
				EPGOpt: config.EPGOpt{
					Disable: false,
					Filters: []config.ArrayMixOpt{
						{
							MixOpt: config.MixOpt{
								SourceName: "non_existent_source",
							},
							FilterBy: string(config.EPGFilterTypeChannelID),
						},
					},
				},
			},
			expectedError: true,
		},
		{
			name: "Multiple filters",
			config: &config.Config{
				EPGOpt: config.EPGOpt{
					Disable: false,
					Filters: []config.ArrayMixOpt{
						{
							MixOpt: config.MixOpt{
								SourceName: "source1",
							},
							FilterBy: string(config.EPGFilterTypeChannelID),
							Include:  "channel1",
						},
						{
							MixOpt: config.MixOpt{
								SourceName: "source2",
							},
							FilterBy: string(config.EPGFilterTypeProgramTitle),
							Include:  "Special",
						},
					},
				},
			},
			expectedCount: 2,
			checkFunction: func(epg *epg.EPG) bool {
				return len(epg.Channel) == 2 &&
					len(epg.Programme) == 2
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟的 Sourcer
			mockSourcer := &MockEPGSourcer{
				sources: map[string]*Source{
					"source1": {
						config: config.Source{
							Type: config.SourceTypeEPG,
						},
						data: []byte(epgData1),
					},
					"source2": {
						config: config.Source{
							Type: config.SourceTypeEPG,
						},
						data: []byte(epgData2),
					},
				},
			}

			// 执行测试
			result, err := MixEPG(tt.config, mockSourcer)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)

			if tt.checkFunction != nil {
				assert.True(t, tt.checkFunction(result))
			}
		})
	}
}

func TestFilterChannels(t *testing.T) {
	channels := []epg.EPGChannel{
		{ID: "channel1"},
		{ID: "channel2"},
		{ID: "channel3"},
	}

	tests := []struct {
		name          string
		filter        config.ArrayMixOpt
		expectedCount int
		expectedIDs   []string
	}{
		{
			name: "Include filter",
			filter: config.ArrayMixOpt{
				FilterBy: string(config.EPGFilterTypeChannelID),
				Include:  "channel[12]",
			},
			expectedCount: 2,
			expectedIDs:   []string{"channel1", "channel2"},
		},
		{
			name: "Exclude filter",
			filter: config.ArrayMixOpt{
				FilterBy: string(config.EPGFilterTypeChannelID),
				Exclude:  "channel2",
			},
			expectedCount: 2,
			expectedIDs:   []string{"channel1", "channel3"},
		},
		{
			name: "Include and Exclude filter",
			filter: config.ArrayMixOpt{
				FilterBy: string(config.EPGFilterTypeChannelID),
				Include:  "channel[12]",
				Exclude:  "channel2",
			},
			expectedCount: 1,
			expectedIDs:   []string{"channel1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterChannels(channels, tt.filter)
			assert.Equal(t, tt.expectedCount, len(result))

			resultIDs := make([]string, len(result))
			for i, channel := range result {
				resultIDs[i] = channel.ID
			}
			assert.ElementsMatch(t, tt.expectedIDs, resultIDs)
		})
	}
}

func TestFilterProgrammes(t *testing.T) {
	programmes := []epg.EPGProgramme{
		{Channel: "channel1", Title: epg.EPGTxt{Text: "Show A"}},
		{Channel: "channel2", Title: epg.EPGTxt{Text: "Show B"}},
		{Channel: "channel3", Title: epg.EPGTxt{Text: "Special Show"}},
	}

	tests := []struct {
		name          string
		filter        config.ArrayMixOpt
		expectedCount int
		checkFunction func([]epg.EPGProgramme) bool
	}{
		{
			name: "Filter by channel ID",
			filter: config.ArrayMixOpt{
				FilterBy: string(config.EPGFilterTypeChannelID),
				Include:  "channel[12]",
			},
			expectedCount: 2,
			checkFunction: func(result []epg.EPGProgramme) bool {
				return result[0].Channel == "channel1" && result[1].Channel == "channel2"
			},
		},
		{
			name: "Filter by program title",
			filter: config.ArrayMixOpt{
				FilterBy: string(config.EPGFilterTypeProgramTitle),
				Include:  "Special",
			},
			expectedCount: 1,
			checkFunction: func(result []epg.EPGProgramme) bool {
				return result[0].Title.Text == "Special Show"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterProgrammes(programmes, tt.filter)
			assert.Equal(t, tt.expectedCount, len(result))

			if tt.checkFunction != nil {
				assert.True(t, tt.checkFunction(result))
			}
		})
	}
}
