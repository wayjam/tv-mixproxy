package mixer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wayjam/tv-mixproxy/config"
)

func TestNewSourceManager(t *testing.T) {
	sources := []config.Source{
		{Name: "test1", URL: "http://example.com/test1", Type: config.SourceTypeSingle, Interval: 60},
		{Name: "test2", URL: "http://example.com/test2", Type: config.SourceTypeMulti, Interval: 120},
	}

	sm := NewSourceManager(sources)

	assert.NotNil(t, sm)
	assert.Len(t, sm.sources, 2)
	assert.NotNil(t, sm.sources["test1"])
	assert.NotNil(t, sm.sources["test2"])
}

func TestGetSource(t *testing.T) {
	// Setup a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		config := config.RepoConfig{
			Spider: "test_spider",
			Sites:  []config.Site{{Key: "test_site", Name: "Test Site"}},
		}
		json.NewEncoder(w).Encode(config)
	}))
	defer server.Close()

	sources := []config.Source{
		{Name: "test", URL: server.URL, Type: config.SourceTypeSingle, Interval: 60},
	}

	sm := NewSourceManager(sources)

	// First call should trigger a refresh
	data, err := sm.GetSource("test")
	assert.NoError(t, err)
	assert.NotNil(t, data)

	var config config.RepoConfig
	err = json.Unmarshal(data.data, &config)
	assert.NoError(t, err)
	assert.Equal(t, "test_spider", config.Spider)
	assert.Len(t, config.Sites, 1)
	assert.Equal(t, "Test Site", config.Sites[0].Name)

	// Second call should return cached data
	data2, err := sm.GetSource("test")
	assert.NoError(t, err)
	assert.Equal(t, data, data2)

	// Test non-existent source
	_, err = sm.GetSource("non_existent")
	assert.Error(t, err)
}

func TestRefreshSource(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		config := config.RepoConfig{
			Spider: "test_spider",
			Sites:  []config.Site{{Key: "test_site", Name: "Test Site"}},
		}
		json.NewEncoder(w).Encode(config)
	}))
	defer server.Close()

	sources := []config.Source{
		{Name: "test", URL: server.URL, Type: config.SourceTypeSingle, Interval: 1}, // 1 second interval
	}

	sm := NewSourceManager(sources)

	// First call
	_, err := sm.GetSource("test")
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)

	// Wait for the interval to pass
	time.Sleep(2 * time.Second)

	// Second call should trigger a refresh
	_, err = sm.GetSource("test")
	assert.NoError(t, err)
	assert.Equal(t, 2, callCount)
}
