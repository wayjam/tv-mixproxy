package server

import (
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wayjam/tv-mixproxy/config"
)

func TestHandlerCachingAnd304(t *testing.T) {
	// Create temporary local M3U file
	tmpDir := t.TempDir()
	m3uPath := filepath.Join(tmpDir, "test.m3u")
	err := os.WriteFile(m3uPath, []byte("#EXTM3U\n#EXTINF:-1 tvg-name=\"CCTV1\",CCTV1\nhttp://example.com/cctv1.m3u8\n"), 0644)
	assert.NoError(t, err)

	cfg := &config.Config{
		ServerPort: 8080,
		Sources: []config.Source{
			{
				Name:     "local_m3u",
				URL:      "file://" + m3uPath,
				Type:     config.SourceTypeM3U,
				Interval: 60,
			},
		},
		M3UOpt: config.M3UOpt{
			Disable: false,
			MediaPlaylistFilters: []config.ArrayMixOpt{
				{MixOpt: config.MixOpt{SourceName: "local_m3u"}},
			},
		},
		TvBoxSingleRepoOpt: config.TvBoxSingleRepoOpt{
			Disable: false,
			Lives: []config.ArrayMixOpt{
				{MixOpt: config.MixOpt{SourceName: "internal/m3u"}},
			},
		},
		EPGOpt: config.EPGOpt{
			Disable: true,
		},
	}
	cfg.Fixture()

	srv := NewServer(cfg)
	srv.SetupRoutes()
	app := srv.App()

	// 1. Test M3U endpoint caching & ETag
	req1 := httptest.NewRequest("GET", "/v1/m3u/media_playlist", nil)
	resp1, err := app.Test(req1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp1.StatusCode)

	body1, _ := io.ReadAll(resp1.Body)
	assert.Contains(t, string(body1), "CCTV1")

	etag := resp1.Header.Get("ETag")
	assert.NotEmpty(t, etag)
	assert.Contains(t, resp1.Header.Get("Cache-Control"), "max-age=60")

	// 2. Test 304 Not Modified with If-None-Match
	req2 := httptest.NewRequest("GET", "/v1/m3u/media_playlist", nil)
	req2.Header.Set("If-None-Match", etag)
	resp2, err := app.Test(req2)
	assert.NoError(t, err)
	assert.Equal(t, 304, resp2.StatusCode)

	// 3. Test TVBox Repo endpoint caching & 304
	reqRepo1 := httptest.NewRequest("GET", "/v1/tvbox/repo", nil)
	respRepo1, err := app.Test(reqRepo1)
	assert.NoError(t, err)
	assert.Equal(t, 200, respRepo1.StatusCode)

	repoEtag := respRepo1.Header.Get("ETag")
	assert.NotEmpty(t, repoEtag)

	reqRepo2 := httptest.NewRequest("GET", "/v1/tvbox/repo", nil)
	reqRepo2.Header.Set("If-None-Match", repoEtag)
	respRepo2, err := app.Test(reqRepo2)
	assert.NoError(t, err)
	assert.Equal(t, 304, respRepo2.StatusCode)

	// 4. Test Live TXT endpoint caching & 304
	reqTxt1 := httptest.NewRequest("GET", "/v1/live.txt", nil)
	respTxt1, err := app.Test(reqTxt1)
	assert.NoError(t, err)
	assert.Equal(t, 200, respTxt1.StatusCode)
	assert.Contains(t, respTxt1.Header.Get("Content-Type"), "text/plain")
	bodyTxt1, _ := io.ReadAll(respTxt1.Body)
	assert.Contains(t, string(bodyTxt1), "CCTV1,http://example.com/cctv1.m3u8")

	txtEtag := respTxt1.Header.Get("ETag")
	assert.NotEmpty(t, txtEtag)

	reqTxt2 := httptest.NewRequest("GET", "/live.txt", nil)
	reqTxt2.Header.Set("If-None-Match", txtEtag)
	respTxt2, err := app.Test(reqTxt2)
	assert.NoError(t, err)
	assert.Equal(t, 304, respTxt2.StatusCode)

	// 5. Test Logo & Wallpaper static cache headers
	reqLogo := httptest.NewRequest("GET", "/logo", nil)
	respLogo, err := app.Test(reqLogo)
	assert.NoError(t, err)
	assert.Equal(t, 200, respLogo.StatusCode)
	assert.Contains(t, respLogo.Header.Get("Cache-Control"), "max-age=86400")
}

func TestDIYPHandlerCachingAnd304(t *testing.T) {
	tmpDir := t.TempDir()
	epgPath := filepath.Join(tmpDir, "test.xml")
	epgXML := `<?xml version="1.0" encoding="UTF-8"?>
<tv>
    <channel id="CCTV1">
        <display-name>CCTV-1 综合</display-name>
    </channel>
    <programme channel="CCTV1" start="20261121083000 +0800" stop="20261121090000 +0800">
        <title>早间新闻</title>
    </programme>
</tv>`
	err := os.WriteFile(epgPath, []byte(epgXML), 0644)
	assert.NoError(t, err)

	cfg := &config.Config{
		ServerPort: 8080,
		Sources: []config.Source{
			{
				Name:     "local_epg",
				URL:      "file://" + epgPath,
				Type:     config.SourceTypeEPG,
				Interval: 60,
			},
		},
		EPGOpt: config.EPGOpt{
			Disable: false,
			Filters: []config.ArrayMixOpt{
				{MixOpt: config.MixOpt{SourceName: "local_epg"}},
			},
		},
		M3UOpt:             config.M3UOpt{Disable: true},
		TvBoxSingleRepoOpt: config.TvBoxSingleRepoOpt{Disable: true},
	}
	cfg.Fixture()

	srv := NewServer(cfg)
	srv.SetupRoutes()
	app := srv.App()

	// 1. Query DIYP EPG
	req1 := httptest.NewRequest("GET", "/api/diyp?ch=CCTV1&date=2026-11-21", nil)
	resp1, err := app.Test(req1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp1.StatusCode)
	assert.Contains(t, resp1.Header.Get("Content-Type"), "application/json")

	body1, _ := io.ReadAll(resp1.Body)
	assert.Contains(t, string(body1), "早间新闻")
	assert.Contains(t, string(body1), "08:30")

	etag := resp1.Header.Get("ETag")
	assert.NotEmpty(t, etag)

	// 2. Query 304 with If-None-Match
	req2 := httptest.NewRequest("GET", "/api/diyp?ch=CCTV1&date=2026-11-21", nil)
	req2.Header.Set("If-None-Match", etag)
	resp2, err := app.Test(req2)
	assert.NoError(t, err)
	assert.Equal(t, 304, resp2.StatusCode)

	// 3. Query via alias route /v1/epg/diyp with fuzzy name
	req3 := httptest.NewRequest("GET", "/v1/epg/diyp?ch=CCTV-1%20%E7%BB%BC%E5%90%88&date=2026-11-21", nil)
	resp3, err := app.Test(req3)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp3.StatusCode)
	body3, _ := io.ReadAll(resp3.Body)
	assert.Contains(t, string(body3), "早间新闻")

	// 4. Query non-existent channel
	req4 := httptest.NewRequest("GET", "/api/diyp?ch=Unknown&date=2026-11-21", nil)
	resp4, err := app.Test(req4)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp4.StatusCode)
	body4, _ := io.ReadAll(resp4.Body)
	assert.Contains(t, string(body4), `"epg_data":[]`)
}
func TestConfigurableCacheAndDisable(t *testing.T) {
	tmpDir := t.TempDir()
	m3uPath := filepath.Join(tmpDir, "test.m3u")
	_ = os.WriteFile(m3uPath, []byte("#EXTM3U\n#EXTINF:-1,CCTV1\nhttp://example.com/cctv1.m3u8\n"), 0644)

	// 1. Test custom max-age = 300
	cfg1 := &config.Config{
		Sources: []config.Source{{Name: "local_m3u", URL: "file://" + m3uPath, Type: config.SourceTypeM3U}},
		M3UOpt: config.M3UOpt{
			MediaPlaylistFilters: []config.ArrayMixOpt{{MixOpt: config.MixOpt{SourceName: "local_m3u"}}},
		},
		CacheOpt: config.CacheOpt{
			MaxAge: 300,
		},
	}
	cfg1.Fixture()
	srv1 := NewServer(cfg1)
	srv1.SetupRoutes()
	resp1, err := srv1.App().Test(httptest.NewRequest("GET", "/v1/m3u/media_playlist", nil))
	assert.NoError(t, err)
	assert.Equal(t, 200, resp1.StatusCode)
	assert.Equal(t, "public, max-age=300", resp1.Header.Get("Cache-Control"))

	// 2. Test cache disabled
	cfg2 := &config.Config{
		Sources: []config.Source{{Name: "local_m3u", URL: "file://" + m3uPath, Type: config.SourceTypeM3U}},
		M3UOpt: config.M3UOpt{
			MediaPlaylistFilters: []config.ArrayMixOpt{{MixOpt: config.MixOpt{SourceName: "local_m3u"}}},
		},
		CacheOpt: config.CacheOpt{
			Disable: true,
		},
	}
	cfg2.Fixture()
	srv2 := NewServer(cfg2)
	srv2.SetupRoutes()
	resp2, err := srv2.App().Test(httptest.NewRequest("GET", "/v1/m3u/media_playlist", nil))
	assert.NoError(t, err)
	assert.Equal(t, 200, resp2.StatusCode)
	assert.Equal(t, "no-cache, no-store, must-revalidate", resp2.Header.Get("Cache-Control"))
}

func TestSecuritySecretRefreshSource(t *testing.T) {
	cfg := &config.Config{
		SecurityOpt: config.SecurityOpt{
			Secret: "custom_pass_123",
		},
	}
	cfg.Fixture()
	srv := NewServer(cfg)
	srv.SetupRoutes()
	app := srv.App()

	// 1. No secret provided -> 401
	req1 := httptest.NewRequest("GET", "/refresh_source", nil)
	resp1, err := app.Test(req1)
	assert.NoError(t, err)
	assert.Equal(t, 401, resp1.StatusCode)

	// 2. Secret in header X-TV-MIXPROXY-SECRET -> 200
	req2 := httptest.NewRequest("GET", "/refresh_source", nil)
	req2.Header.Set("X-TV-MIXPROXY-SECRET", "custom_pass_123")
	resp2, err := app.Test(req2)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp2.StatusCode)

	// 3. Secret in Authorization Bearer -> 200
	req3 := httptest.NewRequest("GET", "/refresh_source", nil)
	req3.Header.Set("Authorization", "Bearer custom_pass_123")
	resp3, err := app.Test(req3)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp3.StatusCode)

	// 4. Secret in query parameter ?token= -> 200
	req4 := httptest.NewRequest("GET", "/refresh_source?token=custom_pass_123", nil)
	resp4, err := app.Test(req4)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp4.StatusCode)
}

func TestAntiOOMAndLRUProtection(t *testing.T) {
	tmpDir := t.TempDir()
	m3uPath := filepath.Join(tmpDir, "test.m3u")
	_ = os.WriteFile(m3uPath, []byte("#EXTM3U\n#EXTINF:-1,CCTV1\nhttp://example.com/cctv1.m3u8\n"), 0644)

	cfg := &config.Config{
		Sources: []config.Source{
			{Name: "local_m3u", URL: "file://" + m3uPath, Type: config.SourceTypeM3U},
		},
		M3UOpt: config.M3UOpt{
			MediaPlaylistFilters: []config.ArrayMixOpt{{MixOpt: config.MixOpt{SourceName: "local_m3u"}}},
		},
		CacheOpt: config.CacheOpt{
			MaxEntries: 3, // Capacity limited to 3 entries
		},
	}
	cfg.Fixture()
	srv := NewServer(cfg)
	srv.SetupRoutes()
	app := srv.App()

	// 1. Initial M3U request (pinned)
	reqM3U := httptest.NewRequest("GET", "/v1/m3u/media_playlist", nil)
	respM3U, err := app.Test(reqM3U)
	assert.NoError(t, err)
	assert.Equal(t, 200, respM3U.StatusCode)
	m3uETag := respM3U.Header.Get("ETag")
	assert.NotEmpty(t, m3uETag)

	// 2. Flood with 10 dynamic DIYP requests
	for i := 0; i < 10; i++ {
		reqFlood := httptest.NewRequest("GET", "/api/diyp?ch=rand_"+string(rune('a'+i)), nil)
		respFlood, err := app.Test(reqFlood)
		assert.NoError(t, err)
		assert.Equal(t, 200, respFlood.StatusCode)
	}

	// 3. Cache length must be bounded by MaxEntries (<= 3)
	assert.LessOrEqual(t, srv.cacheManager.Len(), 3)

	// 4. Pinned M3U must still be in cache and return 304 Not Modified
	reqM3UCheck := httptest.NewRequest("GET", "/v1/m3u/media_playlist", nil)
	reqM3UCheck.Header.Set("If-None-Match", m3uETag)
	respM3UCheck, err := app.Test(reqM3UCheck)
	assert.NoError(t, err)
	assert.Equal(t, 304, respM3UCheck.StatusCode, "pinned m3u must remain in cache despite dynamic flood")
}
