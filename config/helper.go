package config

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// FetchOptions 抓取配置选项
type FetchOptions struct {
	Timeout  time.Duration
	Headers  map[string]string
	ProxyURL string
	Retry    int
}

// FetchDataWithOptions 根据指定的网络选项抓取数据（支持代理、超时、自定义请求头、失败重试）
func FetchDataWithOptions(uri string, opts FetchOptions) ([]byte, error) {
	if strings.HasPrefix(uri, "file://") {
		return os.ReadFile(strings.TrimPrefix(uri, "file://"))
	}

	if !strings.HasPrefix(uri, "http://") && !strings.HasPrefix(uri, "https://") {
		return nil, fmt.Errorf("unsupported URI scheme: %s", uri)
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	if opts.ProxyURL != "" {
		proxy, err := url.Parse(opts.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy url: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxy)
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	retries := opts.Retry
	if retries < 0 {
		retries = 0
	}

	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, uri, nil)
		if err != nil {
			return nil, fmt.Errorf("create request failed: %w", err)
		}

		// 默认伪装常见 User-Agent（防被部分源直接拒绝）
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

		for k, v := range opts.Headers {
			req.Header.Set(k, v)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if resp.StatusCode >= http.StatusBadRequest {
			resp.Body.Close()
			lastErr = fmt.Errorf("failed to fetch data from URL: %s", resp.Status)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read data: %w", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		return data, nil
	}

	return nil, fmt.Errorf("fetch failed after %d attempts: %w", retries+1, lastErr)
}

// FetchData 使用默认选项抓取数据（兼容旧调用）
func FetchData(uri string) ([]byte, error) {
	return FetchDataWithOptions(uri, FetchOptions{Timeout: 30 * time.Second})
}
