package server

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wayjam/tv-mixproxy/config"
)

func TestStreamProxyDisabled(t *testing.T) {
	cfg := &config.Config{
		StreamProxyOpt: config.StreamProxyOpt{Enable: false},
	}
	cfg.Fixture()
	srv := NewServer(cfg)
	srv.SetupRoutes()

	req := httptest.NewRequest("GET", "/v1/stream/proxy?url=http://example.com/stream.m3u8", nil)
	resp, err := srv.App().Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)
}

func TestStreamProxyHeaderForwardingAndTS(t *testing.T) {
	// Mock upstream video streaming server requiring Referer header
	receivedReferer := ""
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReferer = r.Header.Get("Referer")
		if receivedReferer != "https://upstream.example.com/" {
			http.Error(w, "Forbidden hotlinking", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "video/MP2T")
		w.Header().Set("Content-Length", "10")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("0123456789"))
	}))
	defer upstream.Close()

	cfg := &config.Config{
		StreamProxyOpt: config.StreamProxyOpt{
			Enable: true,
		},
		Sources: []config.Source{
			{
				Name: "hotlink_src",
				Headers: map[string]string{
					"Referer": "https://upstream.example.com/",
				},
			},
		},
	}
	cfg.Fixture()
	srv := NewServer(cfg)
	srv.SetupRoutes()
	// 1. Without source specified -> upstream sees no Referer, returns 403
	reqNoRef := httptest.NewRequest("GET", fmt.Sprintf("/v1/stream/proxy?url=%s", url.QueryEscape(upstream.URL+"/stream.ts")), nil)
	respNoRef, err := srv.App().Test(reqNoRef)
	assert.NoError(t, err)
	assert.Equal(t, 403, respNoRef.StatusCode)

	// 2. With source=hotlink_src -> proxy injects Referer header -> upstream returns 200 and binary TS data
	reqWithRef := httptest.NewRequest("GET", fmt.Sprintf("/v1/stream/proxy?url=%s&source=hotlink_src", url.QueryEscape(upstream.URL+"/stream.ts")), nil)
	respWithRef, err := srv.App().Test(reqWithRef)
	assert.NoError(t, err)
	assert.Equal(t, 200, respWithRef.StatusCode)
	assert.Equal(t, "video/MP2T", respWithRef.Header.Get("Content-Type"))

	body, err := io.ReadAll(respWithRef.Body)
	assert.NoError(t, err)
	assert.Equal(t, "0123456789", string(body))
	assert.Equal(t, "https://upstream.example.com/", receivedReferer)
}

func TestStreamProxyM3U8RelativePathRewriting(t *testing.T) {
	m3u8Content := `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:10
#EXTINF:10.0,
chunk-0.ts
#EXTINF:10.0,
chunk-1.ts
#EXT-X-ENDLIST
`
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(m3u8Content))
	}))
	defer upstream.Close()

	cfg := &config.Config{
		StreamProxyOpt: config.StreamProxyOpt{
			Enable: true,
		},
	}
	cfg.Fixture()
	srv := NewServer(cfg)
	srv.SetupRoutes()

	targetURL := upstream.URL + "/live/playlist.m3u8"
	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/stream/proxy?url=%s&source=test_source", url.QueryEscape(targetURL)), nil)
	resp, err := srv.App().Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	bodyStr := string(body)

	// Relative paths chunk-0.ts and chunk-1.ts should be rewritten through /v1/stream/proxy
	assert.Contains(t, bodyStr, "/v1/stream/proxy?url=")
	assert.Contains(t, bodyStr, "source=test_source")
	assert.Contains(t, bodyStr, url.QueryEscape(upstream.URL+"/live/chunk-0.ts"))
	assert.Contains(t, bodyStr, url.QueryEscape(upstream.URL+"/live/chunk-1.ts"))
}

func TestStreamProxyAllowedHosts(t *testing.T) {
	cfg := &config.Config{
		StreamProxyOpt: config.StreamProxyOpt{
			Enable:       true,
			AllowedHosts: []string{"trusted.com", "mycdn.net"},
		},
	}
	cfg.Fixture()
	srv := NewServer(cfg)
	srv.SetupRoutes()

	// 1. Host not allowed -> 403
	req1 := httptest.NewRequest("GET", "/v1/stream/proxy?url=http://evil.com/stream.m3u8", nil)
	resp1, err := srv.App().Test(req1)
	assert.NoError(t, err)
	assert.Equal(t, 403, resp1.StatusCode)
}
