package m3u

import (
	"regexp"
	"strings"
)

var reKeyValue = regexp.MustCompile(`([a-zA-Z0-9_-]+)=("[^"]+"|[^",]+)`)

// DecodeAttributeList turns an attribute list into a key, value map. You should trim
// any characters not part of the attribute list, such as the tag and ':'.
func DecodeAttributeList(line string) map[string]string {
	return decodeParamsLine(line)
}

func decodeParamsLine(line string) map[string]string {
	out := make(map[string]string)
	for _, kv := range reKeyValue.FindAllStringSubmatch(line, -1) {
		k, v := kv[1], kv[2]
		out[k] = strings.Trim(v, ` "`)
	}
	return out
}

func parseAttributes(line string) []Tag {
	var tags []Tag
	line = strings.TrimSpace(line)
	n := len(line)
	i := 0

	for i < n {
		for i < n && (line[i] == ' ' || line[i] == '\t' || line[i] == ',') {
			i++
		}
		if i >= n {
			break
		}

		keyStart := i
		for i < n && line[i] != '=' && line[i] != ' ' && line[i] != '\t' {
			i++
		}
		key := strings.TrimSpace(line[keyStart:i])
		if key == "" {
			continue
		}

		for i < n && (line[i] == ' ' || line[i] == '\t') {
			i++
		}
		if i >= n || line[i] != '=' {
			tags = append(tags, Tag{Name: strings.ToLower(key), Value: ""})
			continue
		}
		i++ // skip '='
		for i < n && (line[i] == ' ' || line[i] == '\t') {
			i++
		}
		if i >= n {
			tags = append(tags, Tag{Name: strings.ToLower(key), Value: ""})
			break
		}

		var val string
		if line[i] == '"' || line[i] == '\'' {
			quote := line[i]
			i++
			valStart := i
			for i < n && line[i] != quote {
				i++
			}
			val = line[valStart:i]
			if i < n && line[i] == quote {
				i++
			}
		} else {
			valStart := i
			for i < n && line[i] != ' ' && line[i] != '\t' {
				i++
			}
			val = line[valStart:i]
		}

		tags = append(tags, Tag{
			Name:  strings.ToLower(key),
			Value: val,
		})
	}

	return tags
}
