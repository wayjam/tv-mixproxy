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
		{Name: "test1", URL: "http://example.com/test1", Type: config.SourceTypeTvBoxSingle, Interval: 60},
		{Name: "test2", URL: "http://example.com/test2", Type: config.SourceTypeTvBoxMulti, Interval: 120},
	}

	sm := NewSourceManager(sources, nil)

	assert.NotNil(t, sm)
	assert.Len(t, sm.sources, 2)
	assert.NotNil(t, sm.sources["test1"])
	assert.NotNil(t, sm.sources["test2"])
}

func TestGetSource(t *testing.T) {
	// Setup a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		config := config.TvBoxRepoConfig{
			Spider: "test_spider",
			Sites:  []config.TvBoxSite{{Key: "test_site", Name: "Test Site"}},
		}
		json.NewEncoder(w).Encode(config)
	}))
	defer server.Close()

	sources := []config.Source{
		{Name: "test", URL: server.URL, Type: config.SourceTypeTvBoxSingle, Interval: 60},
	}

	sm := NewSourceManager(sources, nil)

	// First call should trigger a refresh
	data, err := sm.GetSource("test")
	assert.NoError(t, err)
	assert.NotNil(t, data)

	var config config.TvBoxRepoConfig
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
		config := config.TvBoxRepoConfig{
			Spider: "test_spider",
			Sites:  []config.TvBoxSite{{Key: "test_site", Name: "Test Site"}},
		}
		json.NewEncoder(w).Encode(config)
	}))
	defer server.Close()

	sources := []config.Source{
		{Name: "test", URL: server.URL, Type: config.SourceTypeTvBoxSingle, Interval: 1}, // 1 second interval
	}

	sm := NewSourceManager(sources, nil)

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

func TestConcurrentGetSource_Singleflight(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		time.Sleep(50 * time.Millisecond) // simulate network latency
		config := config.TvBoxRepoConfig{
			Spider: "test_spider_concurrent",
		}
		json.NewEncoder(w).Encode(config)
	}))
	defer server.Close()

	sources := []config.Source{
		{Name: "test_concurrent", URL: server.URL, Type: config.SourceTypeTvBoxSingle, Interval: 60},
	}

	sm := NewSourceManager(sources, nil)

	// Launch 20 concurrent requests
	concurrency := 20
	errCh := make(chan error, concurrency)
	dataCh := make(chan []byte, concurrency)

	for range concurrency {
		go func() {
			src, err := sm.GetSource("test_concurrent")
			if err != nil {
				errCh <- err
				return
			}
			dataCh <- src.Data()
			errCh <- nil
		}()
	}

	for range concurrency {
		err := <-errCh
		assert.NoError(t, err)
		data := <-dataCh
		assert.NotEmpty(t, data)
	}

	// Singleflight should coalesce the 20 requests into 1 network call
	assert.Equal(t, 1, callCount)

	// Verify fingerprint
	fp1 := sm.GetFingerprint("test_concurrent")
	assert.NotEmpty(t, fp1)
	fp2 := sm.GetFingerprint("test_concurrent")
	assert.Equal(t, fp1, fp2)
}

func TestPersistenceAndStaleOnError(t *testing.T) {
	cacheDir := t.TempDir()
	shouldFail := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if shouldFail {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte("source_snapshot_data"))
	}))
	defer server.Close()

	cfg := &config.Config{
		Sources: []config.Source{
			{Name: "test_persist", URL: server.URL, Type: config.SourceTypeM3U, Interval: 60},
		},
		PersistenceOpt: config.PersistenceOpt{
			Enable:       true,
			Dir:          cacheDir,
			StaleOnError: true,
		},
	}
	cfg.Fixture()

	sm := NewSourceManagerWithConfig(cfg, nil)

	// 1. Initial fetch succeeds
	src, err := sm.GetSource("test_persist")
	assert.NoError(t, err)
	assert.Equal(t, "source_snapshot_data", string(src.Data()))

	// 2. Server goes down, manual refresh triggers Stale-on-error fallback
	shouldFail = true
	err = sm.refreshSource("test_persist")
	assert.NoError(t, err, "stale-on-error should suppress error and keep old data")
	assert.Equal(t, "source_snapshot_data", string(src.Data()))

	// 3. Cold restart preloading: create brand new manager with server still down
	smRestart := NewSourceManagerWithConfig(cfg, nil)
	srcRestart, err := smRestart.GetSource("test_persist")
	assert.NoError(t, err)
	assert.Equal(t, "source_snapshot_data", string(srcRestart.Data()))
}
