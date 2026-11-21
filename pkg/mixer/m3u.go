package mixer

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"github.com/wayjam/tv-mixproxy/config"
	"github.com/wayjam/tv-mixproxy/pkg/auth"
	"github.com/wayjam/tv-mixproxy/pkg/cleaner"
	"github.com/wayjam/tv-mixproxy/pkg/m3u"
	"github.com/wayjam/tv-mixproxy/pkg/txt"
)

// MixOptions 混流上下文选项（携带当前请求的用户身份与 Token）
type MixOptions struct {
	User  *auth.User
	Token string
}

func MixM3UMediaPlayList(
	cfg *config.Config, sourcer Sourcer, opts ...MixOptions,
) (*m3u.Playlist, error) {
	if cfg.M3UOpt.Disable {
		return nil, nil
	}

	var user *auth.User
	var token string
	if len(opts) > 0 {
		user = opts[0].User
		token = opts[0].Token
	}
	tokenParam := cfg.SecurityOpt.TokenParam
	if tokenParam == "" {
		tokenParam = "token"
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

			// 权限校验：若当前用户有分组权限限制，过滤非授权分组
			if user != nil && !user.IsGroupAllowed(track.GetTag("group-title")) {
				continue
			}

			// 若开启流代理或需要防盗链，重写流地址为本地代理，并携带当前用户 Token 传播
			if shouldProxy && track.URI != "" {
				proxyURL := fmt.Sprintf("%s/v1/stream/proxy?url=%s&source=%s",
					getExternalURL(cfg),
					url.QueryEscape(track.URI),
					url.QueryEscape(filter.SourceName),
				)
				if token != "" {
					proxyURL = appendToken(proxyURL, token, tokenParam)
				}
				track.URI = proxyURL
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
	// Process EPG URLs injection with token propagation
	if len(cfg.M3UOpt.EPGURLs) > 0 {
		var resolvedEPGs []string
		for _, u := range cfg.M3UOpt.EPGURLs {
			u = strings.ReplaceAll(u, "{external_url}", getExternalURL(cfg))
			if token != "" && strings.Contains(u, getExternalURL(cfg)) {
				u = appendToken(u, token, tokenParam)
			}
			resolvedEPGs = append(resolvedEPGs, u)
		}
		result.SetTag("x-tvg-url", strings.Join(resolvedEPGs, ","))
	} else if existing := result.GetTag("x-tvg-url"); existing != "" {
		if strings.Contains(existing, "{external_url}") {
			existing = strings.ReplaceAll(existing, "{external_url}", getExternalURL(cfg))
			if token != "" && strings.Contains(existing, getExternalURL(cfg)) {
				existing = appendToken(existing, token, tokenParam)
			}
			result.SetTag("x-tvg-url", existing)
		}
	} else if !cfg.EPGOpt.Disable {
		epgURL := getExternalURL(cfg) + "/v1/epg.xml"
		if token != "" {
			epgURL = appendToken(epgURL, token, tokenParam)
		}
		result.SetTag("x-tvg-url", epgURL)
	}
	return &result, nil
}
