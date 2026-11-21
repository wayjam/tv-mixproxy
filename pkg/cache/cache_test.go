package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCacheManager(t *testing.T) {
	cm := NewManager()

	// 1. Get non-existent
	_, ok := cm.Get("key1")
	assert.False(t, ok)

	// 2. Set and Get
	data := []byte("hello world")
	entry := cm.Set("key1", data, "text/plain", "v1")
	assert.NotEmpty(t, entry.ETag)
	assert.Equal(t, "v1", entry.Version)
	assert.Equal(t, "text/plain", entry.ContentType)
	assert.Equal(t, data, entry.Data)

	got, ok := cm.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, entry.ETag, got.ETag)
	assert.Equal(t, data, got.Data)

	// 3. Invalidate
	cm.Invalidate("key1")
	_, ok = cm.Get("key1")
	assert.False(t, ok)

	// 4. Clear
	cm.Set("k1", []byte("1"), "text/plain", "v1")
	cm.Set("k2", []byte("2"), "text/plain", "v1")
	cm.Clear()
	_, ok1 := cm.Get("k1")
	_, ok2 := cm.Get("k2")
	assert.False(t, ok1)
	assert.False(t, ok2)
}

func TestCacheLRUEvictionAndPinned(t *testing.T) {
	opts := Options{
		MaxEntries: 3,
		MaxBytes:   100,
		TTL:        time.Hour,
	}
	cm := NewManagerWithOptions(opts)

	// Set pinned item (e.g. core M3U)
	cm.SetPinned("m3u", []byte("m3u_data"), "text/plain", "v1")
	cm.Set("dyn1", []byte("d1"), "text/plain", "v1")
	cm.Set("dyn2", []byte("d2"), "text/plain", "v1")

	assert.Equal(t, 3, cm.Len())

	// Access dyn1 so dyn2 becomes least recently used
	_, _ = cm.Get("dyn1")

	// Insert dyn3: should evict dyn2, while keeping "m3u" (pinned) and "dyn1"
	cm.Set("dyn3", []byte("d3"), "text/plain", "v1")

	assert.Equal(t, 3, cm.Len())

	_, okM3U := cm.Get("m3u")
	assert.True(t, okM3U, "pinned m3u must not be evicted")

	_, okDyn1 := cm.Get("dyn1")
	assert.True(t, okDyn1, "recently accessed dyn1 must not be evicted")

	_, okDyn2 := cm.Get("dyn2")
	assert.False(t, okDyn2, "dyn2 should have been evicted")

	_, okDyn3 := cm.Get("dyn3")
	assert.True(t, okDyn3)
}

func TestCacheMaxBytesEviction(t *testing.T) {
	opts := Options{
		MaxEntries: 10,
		MaxBytes:   20, // only 20 bytes allowed
		TTL:        time.Hour,
	}
	cm := NewManagerWithOptions(opts)

	cm.Set("k1", []byte("1234567890"), "text/plain", "v1") // 10 bytes
	cm.Set("k2", []byte("1234567890"), "text/plain", "v1") // 10 bytes -> total 20 bytes
	assert.Equal(t, 2, cm.Len())
	assert.Equal(t, int64(20), cm.Bytes())

	// Insert k3 with 10 bytes: exceeds 20 bytes, k1 should be evicted
	cm.Set("k3", []byte("1234567890"), "text/plain", "v1")
	assert.Equal(t, 2, cm.Len())
	assert.Equal(t, int64(20), cm.Bytes())

	_, ok1 := cm.Get("k1")
	assert.False(t, ok1)
	_, ok2 := cm.Get("k2")
	assert.True(t, ok2)
	_, ok3 := cm.Get("k3")
	assert.True(t, ok3)
}

func TestCacheTTLExpiration(t *testing.T) {
	opts := Options{
		MaxEntries: 10,
		MaxBytes:   1000,
		TTL:        10 * time.Millisecond,
	}
	cm := NewManagerWithOptions(opts)

	cm.Set("temp", []byte("data"), "text/plain", "v1")
	_, ok := cm.Get("temp")
	assert.True(t, ok)

	time.Sleep(20 * time.Millisecond)

	_, ok = cm.Get("temp")
	assert.False(t, ok, "entry should have expired")
}

func TestCacheDisabled(t *testing.T) {
	opts := Options{Disable: true}
	cm := NewManagerWithOptions(opts)

	entry := cm.Set("k1", []byte("hello"), "text/plain", "v1")
	assert.NotEmpty(t, entry.ETag)

	_, ok := cm.Get("k1")
	assert.False(t, ok)
	assert.Equal(t, 0, cm.Len())
}
