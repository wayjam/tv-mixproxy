package config

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	defaultHttpClientOnce sync.Once
	defaultHttpClient     *http.Client
)

func GetDefaultHttpClient() *http.Client {
	defaultHttpClientOnce.Do(func() {
		defaultHttpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	})
	return defaultHttpClient
}

func FetchData(uri string) ([]byte, error) {
	var data []byte
	var err error

	if strings.HasPrefix(uri, "file://") {
		// Load from local file
		data, err = os.ReadFile(strings.TrimPrefix(uri, "file://"))
	} else if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		// Load from network URL
		client := GetDefaultHttpClient()
		resp, err := client.Get(uri)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch data from URL: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= http.StatusBadRequest {
			return nil, fmt.Errorf("failed to fetch data from URL: %s", resp.Status)
		}

		data, err = io.ReadAll(resp.Body)

		if err != nil {
			return nil, fmt.Errorf("failed to read data: %v", err)
		}
	} else {
		return nil, fmt.Errorf("unsupported URI scheme: %s", uri)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to read data: %v", err)
	}

	return data, nil
}
