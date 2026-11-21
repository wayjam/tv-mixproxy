package txt

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/wayjam/tv-mixproxy/pkg/m3u"
)

// Channel 表示一个 TXT 直播源中的频道
type Channel struct {
	Group string
	Name  string
	URLs  []string
}

// Playlist 表示包含多个分类和频道的 TXT 列表
type Playlist struct {
	Channels []Channel
}

// Unmarshal 解析 TXT/DIYP 直播源格式
// 支持:
// 央视频道,#genre#
// CCTV-1,http://live1$http://live2
func Unmarshal(data []byte) (*Playlist, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	playlist := &Playlist{
		Channels: make([]Channel, 0),
	}

	currentGroup := "默认频道"

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		lower := strings.ToLower(line)
		if strings.Contains(lower, "#genre") {
			parts := strings.Split(line, ",")
			if len(parts) > 0 {
				grp := strings.TrimSpace(parts[0])
				if grp != "" {
					currentGroup = grp
				}
			}
			continue
		}

		parts := strings.SplitN(line, ",", 2)
		if len(parts) < 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		urlPart := strings.TrimSpace(parts[1])
		if name == "" || urlPart == "" {
			continue
		}

		urlList := strings.Split(urlPart, "$")
		var validURLs []string
		for _, u := range urlList {
			u = strings.TrimSpace(u)
			if u != "" {
				validURLs = append(validURLs, u)
			}
		}

		if len(validURLs) > 0 {
			playlist.Channels = append(playlist.Channels, Channel{
				Group: currentGroup,
				Name:  name,
				URLs:  validURLs,
			})
		}
	}

	return playlist, scanner.Err()
}

// Marshal 将 Playlist 序列化为标准 TXT/DIYP 格式
func (p *Playlist) Marshal() ([]byte, error) {
	var buf bytes.Buffer

	groupMap := make(map[string][]Channel)
	var groupOrder []string

	for _, ch := range p.Channels {
		grp := ch.Group
		if grp == "" {
			grp = "其他频道"
		}
		if _, exists := groupMap[grp]; !exists {
			groupOrder = append(groupOrder, grp)
		}
		groupMap[grp] = append(groupMap[grp], ch)
	}

	for i, grp := range groupOrder {
		buf.WriteString(grp)
		buf.WriteString(",#genre#\n")

		channels := groupMap[grp]
		for _, ch := range channels {
			buf.WriteString(ch.Name)
			buf.WriteString(",")
			buf.WriteString(strings.Join(ch.URLs, "$"))
			buf.WriteString("\n")
		}

		if i < len(groupOrder)-1 {
			buf.WriteString("\n")
		}
	}

	return buf.Bytes(), nil
}

// ToM3U 将 TXT Playlist 转换为 M3U Playlist
func (p *Playlist) ToM3U() *m3u.Playlist {
	m3uPl := m3u.NewPlaylist()

	for _, ch := range p.Channels {
		for _, u := range ch.URLs {
			track := m3u.Track{
				Name:     ch.Name,
				Duration: -1,
				URI:      u,
				Tags: []m3u.Tag{
					{Name: "group-title", Value: ch.Group},
					{Name: "tvg-name", Value: ch.Name},
				},
			}
			m3uPl.Tracks = append(m3uPl.Tracks, track)
		}
	}

	return &m3uPl
}

// FromM3U 从 M3U Playlist 构建 TXT Playlist，相同分组且同名频道的 URL 自动合并为 $
func FromM3U(p *m3u.Playlist) *Playlist {
	playlist := &Playlist{
		Channels: make([]Channel, 0),
	}

	channelIndex := make(map[string]int)

	for _, track := range p.Tracks {
		grp := track.GetTag("group-title")
		if grp == "" {
			grp = "其他频道"
		}
		name := track.Name
		if name == "" {
			name = track.GetTag("tvg-name")
		}
		if name == "" {
			name = "未命名频道"
		}
		uri := track.URI
		if uri == "" {
			continue
		}

		key := grp + "\x00" + name
		if idx, exists := channelIndex[key]; exists {
			alreadyExists := false
			for _, existingURL := range playlist.Channels[idx].URLs {
				if existingURL == uri {
					alreadyExists = true
					break
				}
			}
			if !alreadyExists {
				playlist.Channels[idx].URLs = append(playlist.Channels[idx].URLs, uri)
			}
		} else {
			channelIndex[key] = len(playlist.Channels)
			playlist.Channels = append(playlist.Channels, Channel{
				Group: grp,
				Name:  name,
				URLs:  []string{uri},
			})
		}
	}

	return playlist
}
