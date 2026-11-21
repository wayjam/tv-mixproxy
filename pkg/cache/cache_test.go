package cache

import (
	"testing"

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
