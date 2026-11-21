package m3u

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Track represents an m3u track with a Name, Duration, URI, tags and directives
type Track struct {
	Name       string
	Duration   float64
	URI        string
	Tags       []Tag
	Directives []string
}

func (t *Track) GetTag(name string) string {
	nameLower := strings.ToLower(name)
	for _, tag := range t.Tags {
		if strings.ToLower(tag.Name) == nameLower {
			return tag.Value
		}
	}
	return ""
}

func (t *Track) SetTag(name, value string) {
	nameLower := strings.ToLower(name)
	for i := range t.Tags {
		if strings.ToLower(t.Tags[i].Name) == nameLower {
			t.Tags[i].Value = value
			return
		}
	}
	t.Tags = append(t.Tags, Tag{Name: nameLower, Value: value})
}

func (t *Track) UnmarshalM3U(data []byte) error {
	newTrack, err := parseTrack(string(data))
	if err != nil {
		return err
	}
	*t = *newTrack
	return nil
}

func (track *Track) MarshalM3U() ([]byte, error) {
	buf := new(bytes.Buffer)

	buf.WriteString("#EXTINF:")
	if track.Duration == float64(int64(track.Duration)) {
		buf.WriteString(fmt.Sprintf("%d", int64(track.Duration)))
	} else {
		buf.WriteString(strconv.FormatFloat(track.Duration, 'f', -1, 64))
	}

	for _, tag := range track.Tags {
		buf.WriteString(fmt.Sprintf(" %s=%q", tag.Name, tag.Value))
	}
	buf.WriteString(",")
	buf.WriteString(track.Name)
	buf.WriteRune('\n')

	for _, directive := range track.Directives {
		buf.WriteString(directive)
		buf.WriteRune('\n')
	}

	buf.WriteString(track.URI)
	buf.WriteRune('\n')

	return buf.Bytes(), nil
}

func parseTrack(line string) (*Track, error) {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "#EXTINF:") {
		line = line[8:]
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, errors.New("empty track line")
	}

	endDur := 0
	for endDur < len(line) {
		c := line[endDur]
		if (c >= '0' && c <= '9') || c == '-' || c == '+' || c == '.' {
			endDur++
		} else {
			break
		}
	}
	if endDur == 0 {
		return nil, errors.New("unable to parse duration from EXTINF")
	}

	duration, err := strconv.ParseFloat(line[:endDur], 64)
	if err != nil {
		return nil, fmt.Errorf("unable to parse duration: %w", err)
	}

	rest := line[endDur:]
	if strings.HasPrefix(rest, ",") {
		hasAnotherComma := false
		inQ := false
		for idx := 1; idx < len(rest); idx++ {
			if rest[idx] == '"' || rest[idx] == '\'' {
				inQ = !inQ
			} else if rest[idx] == ',' && !inQ {
				hasAnotherComma = true
				break
			}
		}
		if hasAnotherComma {
			rest = rest[1:]
		}
	}

	var attrPart, titlePart string
	commaIdx := -1
	inQ := false
	for idx := 0; idx < len(rest); idx++ {
		c := rest[idx]
		if c == '"' || c == '\'' {
			inQ = !inQ
		} else if c == ',' && !inQ {
			commaIdx = idx
			break
		}
	}

	if commaIdx != -1 {
		attrPart = rest[:commaIdx]
		titlePart = rest[commaIdx+1:]
	} else {
		titlePart = rest
	}

	track := &Track{
		Name:     strings.TrimSpace(titlePart),
		Duration: duration,
		Tags:     parseAttributes(attrPart),
	}

	return track, nil
}
