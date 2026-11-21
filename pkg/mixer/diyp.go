package mixer

import (
	"strings"
	"time"

	"github.com/wayjam/tv-mixproxy/config"
	"github.com/wayjam/tv-mixproxy/pkg/cleaner"
	"github.com/wayjam/tv-mixproxy/pkg/epg"
)

// DIYPResponse 表示 DIYP 格式的单频道日节目单
type DIYPResponse struct {
	ChannelName string     `json:"channel_name"`
	Date        string     `json:"date"`
	EPGData     []DIYPItem `json:"epg_data"`
}

// DIYPItem 表示单个节目的时间与名称
type DIYPItem struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Title string `json:"title"`
}

// QueryDIYPEPG 查询指定频道在指定日期的节目单
func QueryDIYPEPG(cfg *config.Config, sourcer Sourcer, channelName, dateStr string) (*DIYPResponse, error) {
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	resp := &DIYPResponse{
		ChannelName: channelName,
		Date:        dateStr,
		EPGData:     make([]DIYPItem, 0),
	}

	if cfg.EPGOpt.Disable || channelName == "" {
		return resp, nil
	}

	mixedEPG, err := MixEPG(cfg, sourcer)
	if err != nil {
		return nil, err
	}

	cleanEngine := cleaner.NewEngine(cfg.CleanerOpt, cfg.EPGMappingOpt)
	cleanedName := cleanEngine.CleanName(channelName)

	var targets []string
	targets = append(targets, channelName)
	if cleanedName != channelName {
		targets = append(targets, cleanedName)
	}
	if mappedID, ok := cfg.EPGMappingOpt.Mappings[cleanedName]; ok && mappedID != "" {
		targets = append(targets, mappedID)
	}
	if mappedID, ok := cfg.EPGMappingOpt.Mappings[channelName]; ok && mappedID != "" {
		targets = append(targets, mappedID)
	}

	matchedChannelIDs := findMatchingChannelIDs(mixedEPG.Channel, targets...)
	if len(matchedChannelIDs) == 0 {
		return resp, nil
	}
	targetDateCompact := strings.ReplaceAll(dateStr, "-", "")

	for _, prog := range mixedEPG.Programme {
		if _, ok := matchedChannelIDs[prog.Channel]; !ok {
			continue
		}

		startPart, endPart, matchDate := parseXMLTVTime(prog.Start, prog.Stop)
		if matchDate != targetDateCompact {
			continue
		}

		resp.EPGData = append(resp.EPGData, DIYPItem{
			Start: startPart,
			End:   endPart,
			Title: prog.Title.Text,
		})
	}

	return resp, nil
}

func normalizeChannelName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

func findMatchingChannelIDs(channels []epg.Channel, targets ...string) map[string]struct{} {
	matched := make(map[string]struct{})

	for _, target := range targets {
		targetNorm := normalizeChannelName(target)
		for _, ch := range channels {
			if strings.EqualFold(ch.ID, target) || normalizeChannelName(ch.ID) == targetNorm {
				matched[ch.ID] = struct{}{}
				continue
			}
			for _, dn := range ch.DisplayNames {
				if strings.EqualFold(dn.Text, target) || normalizeChannelName(dn.Text) == targetNorm {
					matched[ch.ID] = struct{}{}
					break
				}
			}
		}
	}

	if len(matched) == 0 {
		for _, target := range targets {
			targetNorm := normalizeChannelName(target)
			for _, ch := range channels {
				for _, dn := range ch.DisplayNames {
					dnNorm := normalizeChannelName(dn.Text)
					if strings.Contains(dnNorm, targetNorm) || strings.Contains(targetNorm, dnNorm) {
						matched[ch.ID] = struct{}{}
						break
					}
				}
			}
		}
	}

	return matched
}

func parseXMLTVTime(startStr, stopStr string) (start, end, dateCompact string) {
	startParts := strings.Fields(startStr)
	if len(startParts) > 0 {
		s := startParts[0]
		if len(s) >= 8 {
			dateCompact = s[:8]
		}
		if len(s) >= 12 {
			start = s[8:10] + ":" + s[10:12]
		}
	}

	stopParts := strings.Fields(stopStr)
	if len(stopParts) > 0 {
		s := stopParts[0]
		if len(s) >= 12 {
			end = s[8:10] + ":" + s[10:12]
		}
	}

	return start, end, dateCompact
}
