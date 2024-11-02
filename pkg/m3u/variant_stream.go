package m3u

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// VariantStream represents an m3u variant stream with a set of attributes
type VariantStream struct {
	Name            string
	URI             string
	Resolution      string
	Bandwidth       int
	AverageBandwith int
	Codecs          string
	FrameRate       float64
	HdcpLevel       string
	Video           string
	Audio           string
	Subtitle        string
	ClosedCaptions  string

	Tags []Tag
}

func (s *VariantStream) UnmarshalM3U(data []byte) error {
	newStream, err := parseVariantStream(string(data))
	if err != nil {
		return err
	}
	*s = *newStream
	return nil
}

func (s *VariantStream) MarshalM3U() ([]byte, error) {
	args := make([]string, 0)

	if s.Resolution != "" {
		args = append(args, fmt.Sprintf("RESOLUTION=%s", s.Resolution))
	}
	if s.Bandwidth > 0 {
		args = append(args, fmt.Sprintf("BANDWIDTH=%d", s.Bandwidth))
	}
	if s.AverageBandwith > 0 {
		args = append(args, fmt.Sprintf("AVERAGE-BANDWIDTH=%d", s.AverageBandwith))
	}
	if s.FrameRate > 0 {
		args = append(args, fmt.Sprintf("FRAME-RATE=%f", s.FrameRate))
	}
	if s.Codecs != "" {
		args = append(args, fmt.Sprintf("CODECS=%s", s.Codecs))
	}
	if s.HdcpLevel != "" {
		args = append(args, fmt.Sprintf("HDCP-LEVEL=%s", s.HdcpLevel))
	}
	if s.Video != "" {
		args = append(args, fmt.Sprintf("VIDEO=%s", s.Video))
	}
	if s.Audio != "" {
		args = append(args, fmt.Sprintf("AUDIO=%s", s.Audio))
	}
	if s.Subtitle != "" {
		args = append(args, fmt.Sprintf("SUBTITLES=%s", s.Subtitle))
	}
	if s.ClosedCaptions != "" {
		args = append(args, fmt.Sprintf("CLOSED-CAPTIONS=%s", s.ClosedCaptions))
	}
	if s.Name != "" {
		args = append(args, fmt.Sprintf("NAME=%s", s.Name))
	}
	if s.URI != "" {
		args = append(args, fmt.Sprintf("URI=%s", s.URI))
	}

	for i := range s.Tags {
		tag := &s.Tags[i]
		args = append(args, fmt.Sprintf("%s=%q", tag.Name, tag.Value))
	}

	return []byte("#EXT-X-STREAM-INF: " + strings.Join(args, ",")), nil
}

func parseVariantStream(line string) (*VariantStream, error) {
	streamInfo := strings.Split(line, ",")
	if len(streamInfo) < 1 {
		return nil, errors.New("invalid m3u file format. Expected EXT-X-STREAM-INF metadata to contain stream data")
	}

	stream := &VariantStream{
		Tags: make([]Tag, 0),
	}

	for _, param := range streamInfo {
		param = strings.TrimSpace(param)
		if err := stream.parseParameter(param); err != nil {
			return nil, err
		}
	}

	return stream, nil
}

func (s *VariantStream) parseParameter(param string) error {
	parts := strings.SplitN(param, "=", 2)
	if len(parts) != 2 {
		return nil // Skip malformed parameters
	}

	key, value := parts[0], parts[1]
	key = strings.TrimSpace(strings.ToUpper(key))
	switch key {
	case "BANDWIDTH":
		bandwidth, err := strconv.Atoi(value)
		if err != nil {
			return errors.New("unable to parse bandwidth")
		}
		s.Bandwidth = bandwidth
	case "AVERAGE-BANDWIDTH":
		avgBandwidth, err := strconv.Atoi(value)
		if err != nil {
			return errors.New("unable to parse average bandwidth")
		}
		s.AverageBandwith = avgBandwidth
	case "RESOLUTION":
		s.Resolution = value
	case "FRAME-RATE":
		frameRate, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return errors.New("unable to parse frame rate")
		}
		s.FrameRate = frameRate
	case "CODECS":
		s.Codecs = value
	case "HDCP-LEVEL":
		s.HdcpLevel = value
	case "VIDEO":
		s.Video = value
	case "AUDIO":
		s.Audio = value
	case "SUBTITLES":
		s.Subtitle = value
	case "CLOSED-CAPTIONS":
		s.ClosedCaptions = value
	case "NAME":
		s.Name = value
	default:
		tag := Tag{
			Name:  key,
			Value: value,
		}
		s.Tags = append(s.Tags, tag)
	}

	return nil
}
