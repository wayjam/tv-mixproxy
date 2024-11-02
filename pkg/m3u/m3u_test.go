package m3u

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestUnmarshalPlaylist(t *testing.T) {
	// 读取测试文件
	data, err := os.ReadFile("testdata/playlist.m3u")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	var playlist Playlist
	err = Unmarshal(data, &playlist)
	if err != nil {
		t.Fatalf("Failed to unmarshal playlist: %v", err)
	}

	// 测试基本属性
	t.Run("Basic Properties", func(t *testing.T) {
		if len(playlist.Tracks) != 13 { // 11 基本tracks + 2 带标签的tracks
			t.Errorf("Expected 13 tracks, got %d", len(playlist.Tracks))
		}
	})

	// 测试基本 track
	t.Run("Basic Track", func(t *testing.T) {
		track := playlist.Tracks[0]
		tests := []struct {
			name     string
			got      interface{}
			expected interface{}
		}{
			{"Length", track.Duration, 10.0},
			{"URI", track.URI, "http://example.com/stream1.ts"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.got != tt.expected {
					t.Errorf("Expected %v, got %v", tt.expected, tt.got)
				}
			})
		}
	})

	// 测试带标签的 track
	t.Run("Track with Tags", func(t *testing.T) {
		// 获取倒数第二个 track（带标签的）
		track := playlist.Tracks[len(playlist.Tracks)-2]

		expectedTags := map[string]string{
			"TVG-ID":      "Channel1",
			"TVG-NAME":    "Channel 1",
			"TVG-LOGO":    "http://example.com/channel1.png",
			"GROUP-TITLE": "Entertainment",
		}

		if track.Name != "Channel 1" {
			t.Errorf("Expected track name to be 'Channel 1', got %s", track.Name)
		}

		if track.URI != "http://example.com/stream1.ts" {
			t.Errorf("Expected URI to be 'http://example.com/stream1.ts', got %s", track.URI)
		}

		foundTags := make(map[string]string)
		for _, tag := range track.Tags {
			foundTags[tag.Name] = tag.Value
		}

		for expectedName, expectedValue := range expectedTags {
			if value, ok := foundTags[expectedName]; !ok {
				t.Errorf("Missing expected tag %s", expectedName)
			} else if value != expectedValue {
				t.Errorf("Tag %s: expected value %s, got %s", expectedName, expectedValue, value)
			}
		}
	})
}

func TestMarshal(t *testing.T) {
	playlist := Playlist{
		Tracks: []Track{
			{
				Name:     "Test Channel",
				Duration: 10,
				URI:      "http://example.com/test.ts",
				Tags: []Tag{
					{Name: "tvg-name", Value: "Test Channel"},
					{Name: "group-title", Value: "Test Group"},
				},
			},
		},
	}

	reader, err := Marshal(&playlist)
	if err != nil {
		t.Fatalf("Failed to marshal playlist: %v", err)
	}

	// 读取生成的内容
	data, err := io.ReadAll(bytes.NewReader(reader))
	if err != nil {
		t.Fatalf("Failed to read marshalled data: %v", err)
	}

	// 验证生成的内容
	expected := `#EXTM3U
#EXTINF:10.000000 tvg-name="Test Channel" group-title="Test Group", Test Channel
http://example.com/test.ts
`

	if string(data) != expected {
		t.Errorf("Marshalled content doesn't match expected.\nExpected:\n%s\nGot:\n%s", expected, string(data))
	}
}
