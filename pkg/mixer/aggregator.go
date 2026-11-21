package mixer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/wayjam/tv-mixproxy/config"
	"github.com/wayjam/tv-mixproxy/pkg/m3u"
)

// TrackCandidate 携带来源优先级与元数据的 Track 候选
type TrackCandidate struct {
	Track         m3u.Track
	Priority      int
	SourceName    string
	OriginalIndex int
}

// Aggregator 负责同名频道去重、优先级排序与多线路合并
type Aggregator struct {
	m3uOpt config.M3UOpt
}

// NewAggregator 创建聚合器
func NewAggregator(m3uOpt config.M3UOpt) *Aggregator {
	return &Aggregator{
		m3uOpt: m3uOpt,
	}
}

// Aggregate 对收集到的 TrackCandidates 执行同名频道去重、优先级排序与多线路合并
func (a *Aggregator) Aggregate(candidates []TrackCandidate) []m3u.Track {
	if len(candidates) == 0 {
		return nil
	}

	if !a.m3uOpt.MergeDuplicateChannels {
		// 未开启同名合并，直接按原样输出（仅剔除完全重复的相同 URI 冗余项）
		seen := make(map[string]struct{})
		var result []m3u.Track
		for _, c := range candidates {
			key := c.Track.Name + "\x00" + c.Track.URI
			if _, exists := seen[key]; !exists {
				seen[key] = struct{}{}
				result = append(result, c.Track)
			}
		}
		return result
	}

	// 开启了同名频道多线路合并
	type channelGroup struct {
		key        string // group + "\x00" + canonicalName
		firstIndex int
		name       string
		group      string
		tracks     []TrackCandidate
	}

	groupMap := make(map[string]*channelGroup)
	var groupOrder []*channelGroup

	for i, c := range candidates {
		name := c.Track.Name
		grp := c.Track.GetTag("group-title")
		key := grp + "\x00" + name

		cg, exists := groupMap[key]
		if !exists {
			cg = &channelGroup{
				key:        key,
				firstIndex: i,
				name:       name,
				group:      grp,
				tracks:     nil,
			}
			groupMap[key] = cg
			groupOrder = append(groupOrder, cg)
		}
		cg.tracks = append(cg.tracks, c)
	}

	var finalTracks []m3u.Track

	for _, cg := range groupOrder {
		// 对同一频道下的候选流按优先级从高到低排序，同优先级保持出现顺序
		sort.SliceStable(cg.tracks, func(i, j int) bool {
			if cg.tracks[i].Priority != cg.tracks[j].Priority {
				return cg.tracks[i].Priority > cg.tracks[j].Priority
			}
			return cg.tracks[i].OriginalIndex < cg.tracks[j].OriginalIndex
		})

		// 剔除相同 URI 的冗余流
		seenURI := make(map[string]struct{})
		var uniqueTracks []TrackCandidate
		for _, tc := range cg.tracks {
			if _, exists := seenURI[tc.Track.URI]; !exists && tc.Track.URI != "" {
				seenURI[tc.Track.URI] = struct{}{}
				uniqueTracks = append(uniqueTracks, tc)
			}
		}

		if len(uniqueTracks) == 0 {
			continue
		}

		if a.m3uOpt.MergeMode == "dollar" {
			// 将多个 URI 用 $ 连接，输出为单个 Track
			baseTrack := uniqueTracks[0].Track
			var uris []string
			for _, ut := range uniqueTracks {
				uris = append(uris, ut.Track.URI)
			}
			baseTrack.URI = strings.Join(uris, "$")
			finalTracks = append(finalTracks, baseTrack)
		} else {
			// 默认 "rename" 模式：如果只有 1 条线路则保持原名，多条线路时标注 (线路1)、(线路2)
			if len(uniqueTracks) == 1 {
				finalTracks = append(finalTracks, uniqueTracks[0].Track)
			} else {
				for lineIdx, ut := range uniqueTracks {
					trackCopy := ut.Track
					trackCopy.Name = fmt.Sprintf("%s (线路%d)", cg.name, lineIdx+1)
					// 保证 tvg-name 依然是原始规范频道名，确保电视端无障碍匹配 EPG 和 Logo
					if trackCopy.GetTag("tvg-name") == "" {
						trackCopy.SetTag("tvg-name", cg.name)
					}
					finalTracks = append(finalTracks, trackCopy)
				}
			}
		}
	}

	return finalTracks
}
