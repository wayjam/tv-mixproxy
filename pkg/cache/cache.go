package cache

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Options 缓存管理器配置选项
type Options struct {
	Disable    bool
	TTL        time.Duration
	MaxEntries int
	MaxBytes   int64
}

// DefaultOptions 返回默认缓存配置
func DefaultOptions() Options {
	return Options{
		Disable:    false,
		TTL:        time.Hour,
		MaxEntries: 1000,
		MaxBytes:   64 * 1024 * 1024, // 64 MB
	}
}

// Entry 表示一个统一缓存条目
type Entry struct {
	Data        []byte
	ContentType string
	ETag        string
	Version     string
	Header      map[string]string
	UpdatedAt   time.Time
	Pinned      bool
}

type cacheItem struct {
	key   string
	entry Entry
}

// Manager 统一内存缓存管理器（支持 LRU 淘汰、TTL 过期与防 OOM 容量限制）
type Manager struct {
	mu           sync.RWMutex
	opts         Options
	items        map[string]*list.Element
	lru          *list.List
	currentBytes int64
}

// NewManager 使用默认配置创建缓存管理器
func NewManager() *Manager {
	return NewManagerWithOptions(DefaultOptions())
}

// NewManagerWithOptions 创建带指定配置的缓存管理器
func NewManagerWithOptions(opts Options) *Manager {
	return &Manager{
		opts:  opts,
		items: make(map[string]*list.Element),
		lru:   list.New(),
	}
}

// Get 获取指定 key 的缓存
func (m *Manager) Get(key string) (Entry, bool) {
	if m.opts.Disable {
		return Entry{}, false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	elem, ok := m.items[key]
	if !ok {
		return Entry{}, false
	}

	item := elem.Value.(*cacheItem)
	if m.opts.TTL > 0 && time.Since(item.entry.UpdatedAt) > m.opts.TTL {
		m.removeElement(elem)
		return Entry{}, false
	}

	m.lru.MoveToFront(elem)
	return item.entry, true
}

// Set 写入指定 key 的缓存，自动计算 ETag
func (m *Manager) Set(key string, data []byte, contentType, version string, headers ...map[string]string) Entry {
	return m.setInternal(key, data, contentType, version, false, headers...)
}

// SetPinned 写入指定 key 的常驻缓存（不被 LRU 挤出）
func (m *Manager) SetPinned(key string, data []byte, contentType, version string, headers ...map[string]string) Entry {
	return m.setInternal(key, data, contentType, version, true, headers...)
}

func (m *Manager) setInternal(key string, data []byte, contentType, version string, pinned bool, headers ...map[string]string) Entry {
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
		Pinned:      pinned,
	}

	if m.opts.Disable {
		return entry
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if elem, ok := m.items[key]; ok {
		item := elem.Value.(*cacheItem)
		m.currentBytes -= int64(len(item.entry.Data))
		item.entry = entry
		m.currentBytes += int64(len(data))
		m.lru.MoveToFront(elem)
		return entry
	}

	m.evictIfNeeded(len(data))

	item := &cacheItem{key: key, entry: entry}
	elem := m.lru.PushFront(item)
	m.items[key] = elem
	m.currentBytes += int64(len(data))

	return entry
}

func (m *Manager) evictIfNeeded(incomingBytes int) {
	if m.opts.MaxEntries <= 0 && m.opts.MaxBytes <= 0 {
		return
	}

	for m.lru.Len() > 0 {
		exceedsEntries := m.opts.MaxEntries > 0 && m.lru.Len() >= m.opts.MaxEntries
		exceedsBytes := m.opts.MaxBytes > 0 && m.currentBytes+int64(incomingBytes) > m.opts.MaxBytes
		if !exceedsEntries && !exceedsBytes {
			break
		}

		// 从链表尾部寻找最久未使用的非常驻条目
		var elemToEvict *list.Element
		for elem := m.lru.Back(); elem != nil; elem = elem.Prev() {
			item := elem.Value.(*cacheItem)
			if !item.entry.Pinned {
				elemToEvict = elem
				break
			}
		}

		// 若所有条目皆为常驻，停止驱逐以避免死循环
		if elemToEvict == nil {
			break
		}

		m.removeElement(elemToEvict)
	}
}

func (m *Manager) removeElement(elem *list.Element) {
	item := elem.Value.(*cacheItem)
	m.lru.Remove(elem)
	delete(m.items, item.key)
	m.currentBytes -= int64(len(item.entry.Data))
	if m.currentBytes < 0 {
		m.currentBytes = 0
	}
}

// Invalidate 使指定 key 缓存失效
func (m *Manager) Invalidate(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if elem, ok := m.items[key]; ok {
		m.removeElement(elem)
	}
}

// Clear 清空全部缓存
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.items = make(map[string]*list.Element)
	m.lru = list.New()
	m.currentBytes = 0
}

// Len 返回当前缓存条目数
func (m *Manager) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.items)
}

// Bytes 返回当前缓存总字节数
func (m *Manager) Bytes() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentBytes
}
