package mixer

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"github.com/wayjam/tv-mixproxy/config"
	"github.com/wayjam/tv-mixproxy/pkg/cleaner"
	"github.com/wayjam/tv-mixproxy/pkg/m3u"
	"github.com/wayjam/tv-mixproxy/pkg/txt"
)

func MixM3UMediaPlayList(
	cfg *config.Config, sourcer Sourcer,
) (*m3u.Playlist, error) {
	if cfg.M3UOpt.Disable {
		return nil, nil
	}

	result := m3u.NewPlaylist()

	if cfg.M3UOpt.MediaPlaylistFallback.SourceName != "" {
		source, err := sourcer.GetSource(cfg.M3UOpt.MediaPlaylistFallback.SourceName)
		if err != nil {
			return nil, fmt.Errorf("get source %s: %w", cfg.M3UOpt.MediaPlaylistFallback.SourceName, err)
		}

		if source.Type() == config.SourceTypeM3U {
			playlist, err := config.ParseM3U8Config(bytes.NewReader(source.Data()))
			if err != nil {
				return nil, fmt.Errorf("parse m3u: %w", err)
			}
			result.Tags = playlist.Tags
			result.Version = playlist.Version
		}
	}
	var candidates []TrackCandidate
	cleanEngine := cleaner.NewEngine(cfg.CleanerOpt, cfg.EPGMappingOpt)

	for _, filter := range cfg.M3UOpt.MediaPlaylistFilters {
		source, err := sourcer.GetSource(filter.SourceName)
		if err != nil {
			return nil, fmt.Errorf("get source %s: %w", filter.SourceName, err)
		}

		priority := 0
		proxyStream := false
		var headers map[string]string
		if source != nil {
			priority = source.Priority()
			proxyStream = source.config.ProxyStream
			headers = source.config.Headers
		}

		var playlist *m3u.Playlist
		switch source.Type() {
		case config.SourceTypeM3U:
			pl, err := config.ParseM3U8Config(bytes.NewReader(source.Data()))
			if err != nil {
				return nil, fmt.Errorf("decode media playlist: %w", err)
			}
			playlist = &pl
		case config.SourceTypeTXT, config.SourceTypeDIYP:
			txtPl, err := txt.Unmarshal(source.Data())
			if err != nil {
				return nil, fmt.Errorf("decode txt playlist: %w", err)
			}
			playlist = txtPl.ToM3U()
		default:
			continue
		}

		includeRegex := compileRegex(filter.Include)
		excludeRegex := compileRegex(filter.Exclude)

		shouldProxy := cfg.StreamProxyOpt.Enable && (proxyStream || (cfg.StreamProxyOpt.AutoProxy && len(headers) > 0))

		for i := range playlist.Tracks {
			track := playlist.Tracks[i]
			targetVal := track.Name
			switch strings.ToLower(filter.FilterBy) {
			case "group-title", "group":
				targetVal = track.GetTag("group-title")
			case "tvg-name":
				targetVal = track.GetTag("tvg-name")
				if targetVal == "" {
					targetVal = track.Name
				}
			case "tvg-id":
				targetVal = track.GetTag("tvg-id")
			case "name", "":
				targetVal = track.Name
			default:
				tagVal := track.GetTag(filter.FilterBy)
				if tagVal != "" {
					targetVal = tagVal
				}
			}

			if !matchFilter(targetVal, includeRegex, excludeRegex) {
				continue
			}

			// 清洗频道名称与标签
			cleanEngine.ProcessTrack(&track)
			if cfg.M3UOpt.DefaultGroup != "" && track.GetTag("group-title") == "" {
				track.SetTag("group-title", cfg.M3UOpt.DefaultGroup)
			}

			// 若开启流代理或需要防盗链，重写流地址为本地代理
			if shouldProxy && track.URI != "" {
				track.URI = fmt.Sprintf("%s/v1/stream/proxy?url=%s&source=%s",
					getExternalURL(cfg),
					url.QueryEscape(track.URI),
					url.QueryEscape(filter.SourceName),
				)
			}

			candidates = append(candidates, TrackCandidate{
				Track:         track,
				Priority:      priority,
				SourceName:    filter.SourceName,
				OriginalIndex: len(candidates),
			})
		}

		for i := range playlist.VariantStreams {
			variant := playlist.VariantStreams[i]
			if filter.FilterBy != "" {
				if !matchFilter(variant.Name, includeRegex, excludeRegex) {
					continue
				}
			}
			result.VariantStreams = append(result.VariantStreams, variant)
		}
	}

	// 执行同名聚合与多线路合并
	aggregator := NewAggregator(cfg.M3UOpt)
	result.Tracks = aggregator.Aggregate(candidates)
	// Process EPG URLs injection
	if len(cfg.M3UOpt.EPGURLs) > 0 {
		var resolvedEPGs []string
		for _, u := range cfg.M3UOpt.EPGURLs {
			u = strings.ReplaceAll(u, "{external_url}", getExternalURL(cfg))
			resolvedEPGs = append(resolvedEPGs, u)
		}
		result.SetTag("x-tvg-url", strings.Join(resolvedEPGs, ","))
	} else if !cfg.EPGOpt.Disable && result.GetTag("x-tvg-url") == "" {
		result.SetTag("x-tvg-url", getExternalURL(cfg)+"/v1/epg.xml")
	}

	return &result, nil
}
