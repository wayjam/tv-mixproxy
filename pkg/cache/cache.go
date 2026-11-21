package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Entry 表示一个统一缓存条目
type Entry struct {
	Data        []byte
	ContentType string
	ETag        string
	Version     string
	Header      map[string]string
	UpdatedAt   time.Time
}

// Manager 统一内存缓存管理器
type Manager struct {
	mu      sync.RWMutex
	entries map[string]Entry
}

// NewManager 创建一个新的缓存管理器
func NewManager() *Manager {
	return &Manager{
		entries: make(map[string]Entry),
	}
}

// Get 获取指定 key 的缓存
func (m *Manager) Get(key string) (Entry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.entries[key]
	return entry, ok
}

// Set 写入指定 key 的缓存，自动计算 ETag
func (m *Manager) Set(key string, data []byte, contentType, version string, headers ...map[string]string) Entry {
	m.mu.Lock()
	defer m.mu.Unlock()

	hash := sha256.Sum256(data)
	etag := fmt.Sprintf(`W/"%s-%x"`, hex.EncodeToString(hash[:8]), len(data))

	var extraHeaders map[string]string
	if len(headers) > 0 {
		extraHeaders = headers[0]
	}

	entry := Entry{
		Data:        data,
		ContentType: contentType,
		ETag:        etag,
		Version:     version,
		Header:      extraHeaders,
		UpdatedAt:   time.Now(),
	}
	m.entries[key] = entry
	return entry
}

// Invalidate 使指定 key 缓存失效
func (m *Manager) Invalidate(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, key)
}

// Clear 清空全部缓存
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = make(map[string]Entry)
}
