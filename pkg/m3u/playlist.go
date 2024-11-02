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
		sep := " "
		if i == len(p.Tags)-1 && i != 0 {
			sep = ""
		}
		writer.Write([]byte(fmt.Sprintf("%s%s=%q", sep, tag.Name, tag.Value)))
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
	switch {
	case strings.HasPrefix(line, "#EXTM3U"):
		p.Tags = p.parseTag(line[7:])
	case strings.HasPrefix(line, "#EXTINF:"):
		track, err := parseTrack(line[8:])
		if err != nil {
			return err
		}
		p.Tracks = append(p.Tracks, *track)
	case strings.HasPrefix(line, "#EXT-X-STREAM-INF:"):
		stream, err := parseVariantStream(line[18:])
		if err != nil {
			return err
		}
		p.VariantStreams = append(p.VariantStreams, *stream)
	case strings.HasPrefix(line, "#EXT-X-VERSION:"):
		version, err := strconv.Atoi(strings.TrimSpace(line[15:]))
		if err != nil {
			return err
		}
		p.Version = version
	case strings.HasPrefix(line, "#") || line == "":
		return nil
	default:
		return p.handleURI(line)
	}
	return nil
}

func (p *Playlist) handleURI(line string) error {
	if len(p.Tracks) == 0 && len(p.VariantStreams) == 0 {
		return errors.New("URI provided for playlist with no tracks or streams")
	}

	uri := strings.Trim(line, " ")
	if len(p.VariantStreams) > 0 {
		p.VariantStreams[len(p.VariantStreams)-1].URI = uri
	} else {
		p.Tracks[len(p.Tracks)-1].URI = uri
	}
	return nil
}

func (p *Playlist) parseTag(line string) []Tag {
	tags := make([]Tag, 0)
	for k, v := range DecodeAttributeList(line) {
		tag := Tag{
			Name:  strings.TrimSpace(strings.ToUpper(k)),
			Value: v,
		}
		tags = append(tags, tag)
	}
	return tags
}
