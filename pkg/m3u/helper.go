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
