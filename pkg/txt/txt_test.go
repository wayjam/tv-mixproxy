package txt

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTXTUnmarshalAndMarshal(t *testing.T) {
	raw := `
央视频道,#genre#
CCTV-1 综合,http://live1.ts$http://live2.ts
CCTV-2 财经,http://cctv2.ts

卫视频道,#genre#
湖南卫视,http://hunan.ts
`

	playlist, err := Unmarshal([]byte(raw))
	assert.NoError(t, err)
	assert.Len(t, playlist.Channels, 3)

	assert.Equal(t, "央视频道", playlist.Channels[0].Group)
	assert.Equal(t, "CCTV-1 综合", playlist.Channels[0].Name)
	assert.Equal(t, []string{"http://live1.ts", "http://live2.ts"}, playlist.Channels[0].URLs)

	assert.Equal(t, "卫视频道", playlist.Channels[2].Group)
	assert.Equal(t, "湖南卫视", playlist.Channels[2].Name)

	marshaled, err := playlist.Marshal()
	assert.NoError(t, err)
	outStr := string(marshaled)

	assert.Contains(t, outStr, "央视频道,#genre#")
	assert.Contains(t, outStr, "CCTV-1 综合,http://live1.ts$http://live2.ts")
	assert.Contains(t, outStr, "卫视频道,#genre#")
	assert.Contains(t, outStr, "湖南卫视,http://hunan.ts")
}

func TestTXTToM3UAndBack(t *testing.T) {
	raw := `
央视频道,#genre#
CCTV-1,http://live1.ts$http://live2.ts
`
	pl, err := Unmarshal([]byte(raw))
	assert.NoError(t, err)

	// Convert to M3U
	m3uPl := pl.ToM3U()
	assert.Len(t, m3uPl.Tracks, 2)
	assert.Equal(t, "CCTV-1", m3uPl.Tracks[0].Name)
	assert.Equal(t, "http://live1.ts", m3uPl.Tracks[0].URI)
	assert.Equal(t, "央视频道", m3uPl.Tracks[0].GetTag("group-title"))
	assert.Equal(t, "CCTV-1", m3uPl.Tracks[1].Name)
	assert.Equal(t, "http://live2.ts", m3uPl.Tracks[1].URI)

	// Convert back from M3U
	txtPl := FromM3U(m3uPl)
	assert.Len(t, txtPl.Channels, 1)
	assert.Equal(t, "CCTV-1", txtPl.Channels[0].Name)
	assert.Equal(t, []string{"http://live1.ts", "http://live2.ts"}, txtPl.Channels[0].URLs)

	out, err := txtPl.Marshal()
	assert.NoError(t, err)
	assert.True(t, strings.Contains(string(out), "CCTV-1,http://live1.ts$http://live2.ts"))
}
