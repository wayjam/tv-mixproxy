package m3u

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func NewPlaylist() Playlist {
	return Playlist{
		Tracks:         []Track{},
		VariantStreams: []VariantStream{},
		Tags:           []Tag{},
	}
}

// Playlist is a type that represents an m3u playlist containing 0 or more tracks or streams
type Playlist struct {
	Tracks         []Track
	VariantStreams []VariantStream
	Version        int
	Tags           []Tag

	currentTrack   *Track
	currentVariant *VariantStream
}

func (p *Playlist) GetTag(name string) string {
	nameLower := strings.ToLower(name)
	for _, tag := range p.Tags {
		if strings.ToLower(tag.Name) == nameLower {
			return tag.Value
		}
	}
	return ""
}

func (p *Playlist) SetTag(name, value string) {
	nameLower := strings.ToLower(name)
	for i := range p.Tags {
		if strings.ToLower(p.Tags[i].Name) == nameLower {
			p.Tags[i].Value = value
			return
		}
	}
	p.Tags = append(p.Tags, Tag{Name: nameLower, Value: value})
}

// A Tag is a key/value pair
type Tag struct {
	Name  string
	Value string
}

// UnmarshalM3U implements the Unmarshaler interface
func (p *Playlist) UnmarshalM3U(data []byte) error {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	return p.unmarshalScanner(scanner)
}

func (p *Playlist) MarshalM3U() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	if err := p.marshalM3U(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *Playlist) marshalM3U(writer io.Writer) error {
	writer.Write([]byte("#EXTM3U"))

	for i := range p.Tags {
		tag := &p.Tags[i]
		writer.Write([]byte(fmt.Sprintf(" %s=%q", tag.Name, tag.Value)))
	}

	writer.Write([]byte("\n"))

	for _, track := range p.Tracks {
		trackBytes, err := track.MarshalM3U()
		if err != nil {
			return err
		}
		writer.Write(trackBytes)
	}

	for _, stream := range p.VariantStreams {
		streamBytes, err := stream.MarshalM3U()
		if err != nil {
			return err
		}
		writer.Write(streamBytes)
	}

	return nil
}

func (p *Playlist) unmarshalScanner(scanner *bufio.Scanner) error {
	onFirstLine := true

	for scanner.Scan() {
		line := scanner.Text()
		if onFirstLine && !strings.HasPrefix(line, "#EXTM3U") {
			return errors.New("invalid m3u file format. Expected #EXTM3U file header")
		}
		onFirstLine = false

		if err := p.parseLine(line); err != nil {
			return err
		}
	}

	return scanner.Err()
}

func (p *Playlist) parseLine(line string) error {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil
	}
	switch {
	case strings.HasPrefix(trimmed, "#EXTM3U"):
		p.Tags = append(p.Tags, p.parseTag(trimmed[7:])...)
	case strings.HasPrefix(trimmed, "#EXTINF:"):
		track, err := parseTrack(trimmed[8:])
		if err != nil {
			return err
		}
		p.Tracks = append(p.Tracks, *track)
		p.currentTrack = &p.Tracks[len(p.Tracks)-1]
		p.currentVariant = nil
	case strings.HasPrefix(trimmed, "#EXT-X-STREAM-INF:"):
		stream, err := parseVariantStream(trimmed[18:])
		if err != nil {
			return err
		}
		p.VariantStreams = append(p.VariantStreams, *stream)
		p.currentVariant = &p.VariantStreams[len(p.VariantStreams)-1]
		p.currentTrack = nil
	case strings.HasPrefix(trimmed, "#EXT-X-VERSION:"):
		version, err := strconv.Atoi(strings.TrimSpace(trimmed[15:]))
		if err != nil {
			return err
		}
		p.Version = version
	case strings.HasPrefix(trimmed, "#EXTGRP:"):
		grp := strings.TrimSpace(trimmed[8:])
		if p.currentTrack != nil {
			p.currentTrack.Directives = append(p.currentTrack.Directives, line)
			if p.currentTrack.GetTag("group-title") == "" {
				p.currentTrack.SetTag("group-title", grp)
			}
		}
	case strings.HasPrefix(trimmed, "#EXTVLCOPT:") || strings.HasPrefix(trimmed, "#KODIPROP:"):
		if p.currentTrack != nil {
			p.currentTrack.Directives = append(p.currentTrack.Directives, line)
		}
	case strings.HasPrefix(trimmed, "#"):
		if p.currentTrack != nil && !strings.HasPrefix(trimmed, "#EXT-X-") {
			p.currentTrack.Directives = append(p.currentTrack.Directives, line)
		}
		return nil
	default:
		return p.handleURI(trimmed)
	}
	return nil
}

func (p *Playlist) handleURI(line string) error {
	uri := strings.TrimSpace(line)
	if p.currentTrack != nil {
		p.currentTrack.URI = uri
		p.currentTrack = nil
		return nil
	}
	if p.currentVariant != nil {
		p.currentVariant.URI = uri
		p.currentVariant = nil
		return nil
	}
	if len(p.Tracks) > 0 && p.Tracks[len(p.Tracks)-1].URI == "" {
		p.Tracks[len(p.Tracks)-1].URI = uri
		return nil
	}
	if len(p.VariantStreams) > 0 && p.VariantStreams[len(p.VariantStreams)-1].URI == "" {
		p.VariantStreams[len(p.VariantStreams)-1].URI = uri
		return nil
	}
	return errors.New("URI provided for playlist with no active tracks or streams")
}

func (p *Playlist) parseTag(line string) []Tag {
	return parseAttributes(line)
}
