package mixer

import (
	"bytes"
	"fmt"
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

	for _, filter := range cfg.M3UOpt.MediaPlaylistFilters {
		source, err := sourcer.GetSource(filter.SourceName)
		if err != nil {
			return nil, fmt.Errorf("get source %s: %w", filter.SourceName, err)
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

			result.Tracks = append(result.Tracks, track)
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
	// Process tracks with cleaner, EPG/logo mapping, and default group
	cleanEngine := cleaner.NewEngine(cfg.CleanerOpt, cfg.EPGMappingOpt)
	for i := range result.Tracks {
		cleanEngine.ProcessTrack(&result.Tracks[i])
		if cfg.M3UOpt.DefaultGroup != "" && result.Tracks[i].GetTag("group-title") == "" {
			result.Tracks[i].SetTag("group-title", cfg.M3UOpt.DefaultGroup)
		}
	}

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
