package mixer

import (
	"bytes"
	"fmt"

	"github.com/wayjam/tv-mixproxy/config"
	"github.com/wayjam/tv-mixproxy/pkg/epg"
)

func MixEPG(
	cfg *config.Config, sourcer Sourcer,
) (*epg.EPG, error) {
	if cfg.EPGOpt.Disable {
		return &epg.EPG{}, nil
	}

	mixedEPG := &epg.EPG{}
	channelMap := make(map[string]epg.EPGChannel) // 用于追踪已添加的频道

	for _, filter := range cfg.EPGOpt.Filters {
		source, err := sourcer.GetSource(filter.SourceName)
		if err != nil {
			return nil, fmt.Errorf("get source %s: %w", filter.SourceName, err)
		}

		if source.Type() != config.SourceTypeEPG {
			continue
		}

		sourceEpg, err := config.ParseEPGConfig(bytes.NewBuffer(source.Data()))
		if err != nil {
			return nil, fmt.Errorf("parse epg config: %w", err)
		}

		// 创建频道ID到频道的映射，用于快速查找
		sourceChannelMap := make(map[string]epg.EPGChannel)
		for _, channel := range sourceEpg.Channel {
			sourceChannelMap[channel.ID] = channel
		}

		if filter.FilterBy == string(config.EPGFilterTypeChannelID) {
			// 按频道ID过滤
			filteredChannels := filterChannels(sourceEpg.Channel, filter)
			for _, channel := range filteredChannels {
				channelMap[channel.ID] = channel
			}
			filteredProgrammes := filterProgrammes(sourceEpg.Programme, filter)
			mixedEPG.Programme = append(mixedEPG.Programme, filteredProgrammes...)
		} else if filter.FilterBy == string(config.EPGFilterTypeProgramTitle) {
			// 按节目标题过滤
			filteredProgrammes := filterProgrammes(sourceEpg.Programme, filter)
			// 收集匹配节目对应的频道
			for _, programme := range filteredProgrammes {
				if channel, exists := sourceChannelMap[programme.Channel]; exists {
					channelMap[channel.ID] = channel
				}
			}
			mixedEPG.Programme = append(mixedEPG.Programme, filteredProgrammes...)
		}
	}

	// 将收集的所有频道添加到最终的EPG中
	for _, channel := range channelMap {
		mixedEPG.Channel = append(mixedEPG.Channel, channel)
	}

	return mixedEPG, nil
}

func filterChannels(channels []epg.EPGChannel, filter config.ArrayMixOpt) []epg.EPGChannel {
	var filtered []epg.EPGChannel
	includeRegex := compileRegex(filter.Include)
	excludeRegex := compileRegex(filter.Exclude)

	for _, channel := range channels {
		if filter.FilterBy == string(config.EPGFilterTypeChannelID) {
			if matchFilter(channel.ID, includeRegex, excludeRegex) {
				filtered = append(filtered, channel)
			}
		}
	}

	return filtered
}

func filterProgrammes(programmes []epg.EPGProgramme, filter config.ArrayMixOpt) []epg.EPGProgramme {
	var filtered []epg.EPGProgramme
	includeRegex := compileRegex(filter.Include)
	excludeRegex := compileRegex(filter.Exclude)

	for _, programme := range programmes {
		if filter.FilterBy == string(config.EPGFilterTypeChannelID) {
			if matchFilter(programme.Channel, includeRegex, excludeRegex) {
				filtered = append(filtered, programme)
			}
		} else if filter.FilterBy == string(config.EPGFilterTypeProgramTitle) {
			if matchFilter(programme.Title.Text, includeRegex, excludeRegex) {
				filtered = append(filtered, programme)
			}
		}
	}

	return filtered
}
