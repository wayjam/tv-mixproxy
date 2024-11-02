package mixer

import (
	"bytes"
	"fmt"

	"github.com/wayjam/tv-mixproxy/config"
	"github.com/wayjam/tv-mixproxy/pkg/m3u"
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

		if source.Type() != config.SourceTypeM3U {
			continue
		}

		playlist, err := config.ParseM3U8Config(bytes.NewReader(source.Data()))
		if err != nil {
			return nil, fmt.Errorf("decode media playlist: %w", err)
		}

		includeRegex := compileRegex(filter.Include)
		excludeRegex := compileRegex(filter.Exclude)

		for i := range playlist.Tracks {
			track := playlist.Tracks[i]
			if filter.FilterBy != "" {
				if !matchFilter(track.Name, includeRegex, excludeRegex) {
					continue
				}
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

	return &result, nil
}
