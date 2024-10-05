package config

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	multiRepoTestData = `
	// this is the comment
{
	"urls": [

		{
			"url": "http://example.com/api.json",
			"name": "repo1"
		},
		{
			"url": "https://example.com/all.json",
			"name": "repo2"
		}
	]
}`
)

func TestLoadMultiRepoConfig(t *testing.T) {
	t.Run("Load from local file", func(t *testing.T) {
		// Create a temporary file with test data
		tempDir := t.TempDir()
		tempFile := filepath.Join(tempDir, "test_config.json")
		err := os.WriteFile(tempFile, []byte(multiRepoTestData), 0644)
		assert.NoError(t, err)

		// Test loading from local file
		config, err := LoadMultiRepoConfig("file://" + tempFile)
		assert.NoError(t, err)
		assert.NotNil(t, config)
		assert.Len(t, config.Repos, 2)
		assert.Equal(t, "repo1", config.Repos[0].Name)
		assert.Equal(t, "http://example.com/api.json", config.Repos[0].URL)
	})

	t.Run("Load from network URL", func(t *testing.T) {
		// Create a test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, multiRepoTestData)
		}))
		defer server.Close()

		// Test loading from network URL
		config, err := LoadMultiRepoConfig(server.URL)
		assert.NoError(t, err)
		assert.NotNil(t, config)
		assert.Len(t, config.Repos, 2)
		assert.Equal(t, "repo2", config.Repos[1].Name)
		assert.Equal(t, "https://example.com/all.json", config.Repos[1].URL)
	})

	t.Run("Unsupported URI scheme", func(t *testing.T) {
		_, err := LoadMultiRepoConfig("ftp://example.com/config.json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported URI scheme")
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		tempDir := t.TempDir()
		tempFile := filepath.Join(tempDir, "invalid_config.json")
		invalidData := `{"urls": [{"url": "http://example.com", "name": "Test Repo"}]` // Missing closing brace
		err := os.WriteFile(tempFile, []byte(invalidData), 0644)
		assert.NoError(t, err)

		_, err = LoadMultiRepoConfig("file://" + tempFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse JSON")
	})
}

var (
	tvBoxTestData = `
	// this is a comment
	{
		"spider": "https://example.com/spider.jar",
		"wallpaper": "https://example.com/wallpaper.jpg",
		"sites": [
			{
				"key": "site1",
				"name": "Site 1",
				"type": 3,
				"api": "https://example.com/api1",
				"searchable": 1,
				"quickSearch": 1,
				"filterable": 1
			},
			{
				"key": "site2",
				"name": "Site 2",
				"type": 1,
				"api": "https://example.com/api2",
				"searchable": 0,
				"quickSearch": 0,
				"filterable": 0,
				"playerType": 2
			}
		],
		"lives": [
			{
				"name": "Live Channel",
				"type": 0,
				"url": "https://example.com/live.m3u",
				"playerType": 1,
				"ua": "Mozilla/5.0",
				"epg": "https://example.com/epg",
				"logo": "https://example.com/logo.png"
			}
		],
		"doh": [
			{
				"name": "Google",
				"url": "https://dns.google/dns-query",
				"ips": ["8.8.8.8", "8.8.4.4"]
			}
		],
		"parses": [
			{
				"name": "Parser 1",
				"type": 0,
				"url": "https://example.com/parser1"
			}
		],
		"flags": ["flag1", "flag2"],
		"rules": [
			{
				"name": "Rule 1",
				"hosts": ["example.com"],
				"regex": ["pattern1", "pattern2"]
			}
		],
		"ads": ["ad1", "ad2"]
	}`
)

func TestLoadTvBoxConfig(t *testing.T) {
	t.Run("Load from local file", func(t *testing.T) {
		// Create a temporary file with test data
		tempDir := t.TempDir()
		tempFile := filepath.Join(tempDir, "test_tvbox_config.json")
		err := os.WriteFile(tempFile, []byte(tvBoxTestData), 0644)
		assert.NoError(t, err)

		// Test loading from local file
		config, err := LoadTvBoxConfig("file://" + tempFile)
		assert.NoError(t, err)
		assert.NotNil(t, config)

		// Verify the loaded config
		assert.Equal(t, "https://example.com/spider.jar", config.Spider)
		assert.Equal(t, "https://example.com/wallpaper.jpg", config.Wallpaper)
		assert.Len(t, config.Sites, 2)
		assert.Equal(t, "site1", config.Sites[0].Key)
		assert.Equal(t, FlexInt(2), config.Sites[1].PlayerType)
		assert.Len(t, config.DOH, 1)
		assert.Equal(t, "Google", config.DOH[0].Name)
		assert.Len(t, config.Lives, 1)
		assert.Equal(t, "Live Channel", config.Lives[0].Name)
		assert.Equal(t, "https://example.com/epg", config.Lives[0].EPG)
		assert.Equal(t, "https://example.com/logo.png", config.Lives[0].Logo)
		assert.Len(t, config.Parses, 1)
		assert.Equal(t, "Parser 1", config.Parses[0].Name)
		assert.Equal(t, []string{"flag1", "flag2"}, config.Flags)
		assert.Len(t, config.Rules, 1)
		assert.Equal(t, "Rule 1", config.Rules[0].Name)
		assert.Equal(t, []string{"ad1", "ad2"}, config.Ads)
	})

	t.Run("Load from network URL", func(t *testing.T) {
		// Create a test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, tvBoxTestData)
		}))
		defer server.Close()

		// Test loading from network URL
		config, err := LoadTvBoxConfig(server.URL)
		assert.NoError(t, err)
		assert.NotNil(t, config)

		// Verify the loaded config
		assert.Equal(t, "https://example.com/spider.jar", config.Spider)
		assert.Len(t, config.Sites, 2)
		assert.Equal(t, "site2", config.Sites[1].Key)
		assert.Len(t, config.DOH, 1)
		assert.Equal(t, []string{"8.8.8.8", "8.8.4.4"}, config.DOH[0].IPs)
		assert.Len(t, config.Lives, 1)
		assert.Equal(t, "Mozilla/5.0", config.Lives[0].UA)
	})

	t.Run("Unsupported URI scheme", func(t *testing.T) {
		_, err := LoadTvBoxConfig("ftp://example.com/config.json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported URI scheme")
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		tempDir := t.TempDir()
		tempFile := filepath.Join(tempDir, "invalid_tvbox_config.json")
		invalidData := `{"spider": "https://example.com/spider.jar", "sites": [{"key": "site1"}]` // Missing closing brace
		err := os.WriteFile(tempFile, []byte(invalidData), 0644)
		assert.NoError(t, err)

		_, err = LoadTvBoxConfig("file://" + tempFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse JSON")
	})
}
