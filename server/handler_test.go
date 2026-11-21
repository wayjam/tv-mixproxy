package server

import (
	"encoding/json"
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
func TestAuthMiddleware_SingleKey(t *testing.T) {
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
		SecurityOpt: config.SecurityOpt{
			Enable:     true,
			AccessKey:  "my_single_secret",
			TokenParam: "token",
			HeaderName: "X-Access-Key",
		},
	}
	cfg.Fixture()
	srv := NewServer(cfg)
	srv.SetupRoutes()
	app := srv.App()

	// 1. Unauthorized anonymous request -> 401
	req1 := httptest.NewRequest("GET", "/v1/m3u/media_playlist", nil)
	resp1, err := app.Test(req1)
	assert.NoError(t, err)
	assert.Equal(t, 401, resp1.StatusCode)

	// 2. Wrong token -> 401
	req2 := httptest.NewRequest("GET", "/v1/m3u/media_playlist?token=wrong_key", nil)
	resp2, err := app.Test(req2)
	assert.NoError(t, err)
	assert.Equal(t, 401, resp2.StatusCode)

	// 3. Query string token -> 200
	req3 := httptest.NewRequest("GET", "/v1/m3u/media_playlist?token=my_single_secret", nil)
	resp3, err := app.Test(req3)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp3.StatusCode)

	// 4. Custom header X-Access-Key -> 200
	req4 := httptest.NewRequest("GET", "/v1/m3u/media_playlist", nil)
	req4.Header.Set("X-Access-Key", "my_single_secret")
	resp4, err := app.Test(req4)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp4.StatusCode)

	// 5. Authorization Bearer -> 200
	req5 := httptest.NewRequest("GET", "/v1/m3u/media_playlist", nil)
	req5.Header.Set("Authorization", "Bearer my_single_secret")
	resp5, err := app.Test(req5)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp5.StatusCode)

	// 6. Whitelist bypass for /logo and /wallpaper without token -> 200
	reqLogo := httptest.NewRequest("GET", "/logo", nil)
	respLogo, err := app.Test(reqLogo)
	assert.NoError(t, err)
	assert.Equal(t, 200, respLogo.StatusCode)

	reqWall := httptest.NewRequest("GET", "/wallpaper", nil)
	respWall, err := app.Test(reqWall)
	assert.NoError(t, err)
	assert.Equal(t, 200, respWall.StatusCode)
}

func TestAuthMiddleware_MultiUser_GroupFiltering_CacheIsolation(t *testing.T) {
	tmpDir := t.TempDir()
	m3uPath := filepath.Join(tmpDir, "test.m3u")
	m3uContent := `#EXTM3U
#EXTINF:-1 group-title="央视",CCTV-1 综合
http://example.com/cctv1.m3u8
#EXTINF:-1 group-title="少儿",卡酷少儿
http://example.com/kaku.m3u8
#EXTINF:-1 group-title="电影",动作电影
http://example.com/action.m3u8
`
	_ = os.WriteFile(m3uPath, []byte(m3uContent), 0644)

	disabledBool := false
	cfg := &config.Config{
		Sources: []config.Source{
			{Name: "local_m3u", URL: "file://" + m3uPath, Type: config.SourceTypeM3U},
		},
		M3UOpt: config.M3UOpt{
			MediaPlaylistFilters: []config.ArrayMixOpt{{MixOpt: config.MixOpt{SourceName: "local_m3u"}}},
		},
		SecurityOpt: config.SecurityOpt{
			Enable: true,
			Users: []config.UserOpt{
				{ID: "admin", Key: "key_admin"},
				{ID: "kids", Key: "key_kids", AllowedGroups: []string{"少儿"}},
				{ID: "cctv", Key: "key_cctv", AllowedGroups: []string{"央视"}},
				{ID: "dis", Key: "key_dis", Enabled: &disabledBool},
				{ID: "exp", Key: "key_exp", ExpireAt: "2020-01-01T00:00:00Z"},
			},
		},
	}
	cfg.Fixture()
	srv := NewServer(cfg)
	srv.SetupRoutes()
	app := srv.App()

	// 1. Disabled user rejected
	reqDis := httptest.NewRequest("GET", "/v1/m3u/media_playlist?token=key_dis", nil)
	respDis, err := app.Test(reqDis)
	assert.NoError(t, err)
	assert.Equal(t, 401, respDis.StatusCode)

	// 2. Expired user rejected
	reqExp := httptest.NewRequest("GET", "/v1/m3u/media_playlist?token=key_exp", nil)
	respExp, err := app.Test(reqExp)
	assert.NoError(t, err)
	assert.Equal(t, 401, respExp.StatusCode)

	// 3. Kids user gets only "少儿" channels
	reqKids := httptest.NewRequest("GET", "/v1/m3u/media_playlist?token=key_kids", nil)
	respKids, err := app.Test(reqKids)
	assert.NoError(t, err)
	assert.Equal(t, 200, respKids.StatusCode)
	bodyKids, _ := io.ReadAll(respKids.Body)
	kidsStr := string(bodyKids)
	assert.Contains(t, kidsStr, "卡酷少儿")
	assert.NotContains(t, kidsStr, "CCTV-1")
	assert.NotContains(t, kidsStr, "动作电影")

	// 4. CCTV user gets only "央视" channels
	reqCCTV := httptest.NewRequest("GET", "/v1/m3u/media_playlist?token=key_cctv", nil)
	respCCTV, err := app.Test(reqCCTV)
	assert.NoError(t, err)
	assert.Equal(t, 200, respCCTV.StatusCode)
	bodyCCTV, _ := io.ReadAll(respCCTV.Body)
	cctvStr := string(bodyCCTV)
	assert.Contains(t, cctvStr, "CCTV-1")
	assert.NotContains(t, cctvStr, "卡酷少儿")
	assert.NotContains(t, cctvStr, "动作电影")

	// 5. Admin user gets all channels
	reqAdmin := httptest.NewRequest("GET", "/v1/m3u/media_playlist?token=key_admin", nil)
	respAdmin, err := app.Test(reqAdmin)
	assert.NoError(t, err)
	assert.Equal(t, 200, respAdmin.StatusCode)
	bodyAdmin, _ := io.ReadAll(respAdmin.Body)
	adminStr := string(bodyAdmin)
	assert.Contains(t, adminStr, "CCTV-1")
	assert.Contains(t, adminStr, "卡酷少儿")
	assert.Contains(t, adminStr, "动作电影")

	// 6. Verify cache segregation: keys are separate and not contaminated
	entryKids, okKids := srv.cacheManager.Get("m3u_media_playlist_kids")
	assert.True(t, okKids)
	assert.NotContains(t, string(entryKids.Data), "CCTV-1")

	entryAdmin, okAdmin := srv.cacheManager.Get("m3u_media_playlist_admin")
	assert.True(t, okAdmin)
	assert.Contains(t, string(entryAdmin.Data), "CCTV-1")
}

func TestTokenSelfPropagation(t *testing.T) {
	tmpDir := t.TempDir()
	m3uPath := filepath.Join(tmpDir, "test.m3u")
	m3uContent := `#EXTM3U
#EXTINF:-1,Stream1
#EXTVLCOPT:http-user-agent=CustomUA
http://upstream.live/test.m3u8
`
	_ = os.WriteFile(m3uPath, []byte(m3uContent), 0644)

	cfg := &config.Config{
		ServerPort: 8080,
		Sources: []config.Source{
			{Name: "local_m3u", URL: "file://" + m3uPath, Type: config.SourceTypeM3U, ProxyStream: true},
		},
		M3UOpt: config.M3UOpt{
			MediaPlaylistFilters: []config.ArrayMixOpt{{MixOpt: config.MixOpt{SourceName: "local_m3u"}}},
		},
		StreamProxyOpt: config.StreamProxyOpt{
			Enable:    true,
			AutoProxy: true,
		},
		TvBoxSingleRepoOpt: config.TvBoxSingleRepoOpt{
			Disable: false,
			Lives: []config.ArrayMixOpt{
				{MixOpt: config.MixOpt{SourceName: "internal/m3u"}},
			},
		},
		TvBoxMultiRepoOpt: config.TvBoxMultiRepoOpt{
			Disable:           false,
			IncludeSingleRepo: true,
		},
		SecurityOpt: config.SecurityOpt{
			Enable:     true,
			AccessKey:  "my_prop_token",
			TokenParam: "token",
		},
	}
	cfg.Fixture()
	srv := NewServer(cfg)
	srv.SetupRoutes()
	app := srv.App()

	// 1. M3U playlist contains token in x-tvg-url and stream proxy url
	reqM3U := httptest.NewRequest("GET", "/v1/m3u/media_playlist?token=my_prop_token", nil)
	respM3U, err := app.Test(reqM3U)
	assert.NoError(t, err)
	assert.Equal(t, 200, respM3U.StatusCode)
	bodyM3U, _ := io.ReadAll(respM3U.Body)
	m3uStr := string(bodyM3U)

	assert.Contains(t, m3uStr, "x-tvg-url=\"http://localhost:8080/v1/epg.xml?token=my_prop_token\"")
	assert.Contains(t, m3uStr, "/v1/stream/proxy?url=")
	assert.Contains(t, m3uStr, "&token=my_prop_token")

	// 2. TVBox repo contains token in spider, lives url, lives epg
	reqRepo := httptest.NewRequest("GET", "/v1/tvbox/repo?token=my_prop_token", nil)
	respRepo, err := app.Test(reqRepo)
	assert.NoError(t, err)
	assert.Equal(t, 200, respRepo.StatusCode)
	bodyRepo, _ := io.ReadAll(respRepo.Body)
	var repoData config.TvBoxRepoConfig
	err = json.Unmarshal(bodyRepo, &repoData)
	assert.NoError(t, err)
	assert.Contains(t, repoData.Spider, "token=my_prop_token")
	assert.NotEmpty(t, repoData.Lives)
	assert.Contains(t, repoData.Lives[0].URL, "token=my_prop_token")
	assert.Contains(t, repoData.Lives[0].EPG, "token=my_prop_token")

	// 3. Multi repo contains token in repo url
	reqMulti := httptest.NewRequest("GET", "/v1/tvbox/multi_repo?token=my_prop_token", nil)
	respMulti, err := app.Test(reqMulti)
	assert.NoError(t, err)
	assert.Equal(t, 200, respMulti.StatusCode)
	bodyMulti, _ := io.ReadAll(respMulti.Body)
	var multiData config.TvBoxMultiRepoConfig
	err = json.Unmarshal(bodyMulti, &multiData)
	assert.NoError(t, err)
	assert.NotEmpty(t, multiData.Repos)
	assert.Contains(t, multiData.Repos[0].URL, "token=my_prop_token")

	// 4. Stream proxy requires token when security is enabled
	reqProxyNoToken := httptest.NewRequest("GET", "/v1/stream/proxy?url=http%3A%2F%2Fupstream.live%2Ftest.m3u8", nil)
	respProxyNoToken, err := app.Test(reqProxyNoToken)
	assert.NoError(t, err)
	assert.Equal(t, 401, respProxyNoToken.StatusCode)
}
