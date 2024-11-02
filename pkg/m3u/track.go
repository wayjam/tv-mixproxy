package m3u

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Track represents an m3u track with a Name, Lengh, URI and a set of tags
type Track struct {
	Name     string
	Duration float64
	URI      string
	Tags     []Tag
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
	buf.WriteString(fmt.Sprintf("%f ", track.Duration))
	for i := range track.Tags {
		tag := &track.Tags[i]
		sep := " "
		if i == len(track.Tags)-1 {
			sep = ""
		}
		buf.WriteString(fmt.Sprintf("%s=%q%s", tag.Name, tag.Value, sep))
	}
	buf.WriteString(", ")

	buf.WriteString(track.Name)
	buf.WriteRune('\n')
	buf.WriteString(track.URI)
	buf.WriteRune('\n')

	return buf.Bytes(), nil
}

func parseTrack(line string) (*Track, error) {
	trackInfo := strings.Split(line, ",")
	if len(trackInfo) < 2 {
		return nil, errors.New("invalid m3u file format. EXTINF should contain track duration and name")
	}

	duration, err := strconv.ParseFloat(strings.Split(trackInfo[0], " ")[0], 64)
	if err != nil {
		return nil, errors.New("unable to parse duration")
	}

	track := &Track{
		Name:     strings.Trim(trackInfo[len(trackInfo)-1], " "),
		Duration: duration,
		Tags:     make([]Tag, 0),
	}

	for k, v := range DecodeAttributeList(line) {
		tag := Tag{
			Name:  strings.TrimSpace(strings.ToUpper(k)),
			Value: v,
		}
		track.Tags = append(track.Tags, tag)
	}

	return track, nil
}
