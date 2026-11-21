package server

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/wayjam/tv-mixproxy/config"
	"github.com/wayjam/tv-mixproxy/pkg/mixer"
)

// NewStreamProxyHandler 创建透明流代理处理函数，自动注入上游所需的防盗链请求头
func NewStreamProxyHandler(cfg *config.Config, sourceManager *mixer.SourceManager) fiber.Handler {
	timeout := 15 * time.Second
	if cfg != nil && cfg.StreamProxyOpt.Timeout > 0 {
		timeout = time.Duration(cfg.StreamProxyOpt.Timeout) * time.Second
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	if cfg != nil && cfg.NetworkOpt.ProxyURL != "" {
		if proxyURL, err := url.Parse(cfg.NetworkOpt.ProxyURL); err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	httpClient := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	return func(c fiber.Ctx) error {
		if cfg != nil && !cfg.StreamProxyOpt.Enable {
			return c.Status(fiber.StatusForbidden).SendString("Stream proxy is disabled")
		}

		targetURLStr := c.Query("url")
		if targetURLStr == "" {
			if b64 := c.Query("b64"); b64 != "" {
				if dec, err := base64.URLEncoding.DecodeString(b64); err == nil {
					targetURLStr = string(dec)
				}
			}
		}

		if targetURLStr == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Missing target stream url")
		}

		targetURL, err := url.Parse(targetURLStr)
		if err != nil || (targetURL.Scheme != "http" && targetURL.Scheme != "https") {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid stream url")
		}

		// 域名白名单校验
		if cfg != nil && len(cfg.StreamProxyOpt.AllowedHosts) > 0 {
			allowed := false
			for _, h := range cfg.StreamProxyOpt.AllowedHosts {
				if strings.EqualFold(h, targetURL.Hostname()) || strings.HasSuffix(strings.ToLower(targetURL.Hostname()), "."+strings.ToLower(h)) {
					allowed = true
					break
				}
			}
			if !allowed {
				return c.Status(fiber.StatusForbidden).SendString("Host not allowed by stream proxy policy")
			}
		}

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, targetURL.String(), nil)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		// 注入默认请求头
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		if cfg != nil {
			for k, v := range cfg.NetworkOpt.Headers {
				req.Header.Set(k, v)
			}
		}

		// 注入源特定请求头（防盗链 Referer/Origin 等）
		sourceName := c.Query("source")
		if sourceName != "" && sourceManager != nil {
			if src, err := sourceManager.GetSource(sourceName); err == nil && src != nil {
				cfgSrc := src.Config()
				for k, v := range cfgSrc.Headers {
					req.Header.Set(k, v)
				}
			}
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return c.Status(fiber.StatusBadGateway).SendString(fmt.Sprintf("Failed to fetch stream: %v", err))
		}

		contentType := resp.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		// 如果是 M3U8 播放列表，重写其中的相对切片 URL，保证二级切片请求同样经过代理
		isM3U8 := strings.Contains(strings.ToLower(contentType), "mpegurl") || strings.HasSuffix(strings.ToLower(targetURL.Path), ".m3u8")
		if isM3U8 && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return c.Status(fiber.StatusBadGateway).SendString("Failed to read m3u8 playlist")
			}

			rewritten := rewriteM3U8Playlist(string(bodyBytes), targetURL, sourceName)
			c.Set("Content-Type", contentType)
			c.Status(resp.StatusCode)
			return c.SendString(rewritten)
		}

		// 针对 TS 切片或直链视频流，采用 SendStream 零内存积压流式转发
		c.Set("Content-Type", contentType)
		if cl := resp.Header.Get("Content-Length"); cl != "" {
			c.Set("Content-Length", cl)
		}
		c.Status(resp.StatusCode)

		return c.SendStream(resp.Body)
	}
}

func rewriteM3U8Playlist(content string, baseURL *url.URL, sourceName string) string {
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		rawLine := scanner.Text()
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			lines = append(lines, rawLine)
			continue
		}

		resolvedURL := baseURL
		if u, err := url.Parse(line); err == nil {
			resolvedURL = baseURL.ResolveReference(u)
		} else {
			lines = append(lines, line)
			continue
		}

		proxySegURL := fmt.Sprintf("/v1/stream/proxy?url=%s", url.QueryEscape(resolvedURL.String()))
		if sourceName != "" {
			proxySegURL += "&source=" + url.QueryEscape(sourceName)
		}
		lines = append(lines, proxySegURL)
	}
	return strings.Join(lines, "\n")
}
