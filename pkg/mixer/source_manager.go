package mixer

import (
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wayjam/tv-mixproxy/config"
)

type Sourcer interface {
	GetSource(name string) (*Source, error)
}

// type SingleSourcer interface {
// 	Sourcer
// 	Type() config.SourceType
// 	URL() string
// 	Name() string
// }

// var (
// 	_ Sourcer = &SourceManager{}
// 	_ Sourcer = &Source{}
// )

type SourceManager struct {
	cfg     *config.Config
	sources map[string]*Source
	mu      sync.RWMutex
	ticker  *time.Ticker
	done    chan bool
	refresh chan bool
	logger  *slog.Logger
}

type Source struct {
	config     config.Source
	lastUpdate time.Time
	data       []byte
	lastError  time.Time
	errorCount int
	version    int64
	refreshing bool
	refreshCh  chan struct{}
	mu         sync.RWMutex
}

func (s *Source) Data() []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data
}

func (s *Source) Version() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}

func (s *Source) LastUpdate() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastUpdate
}
func (s *Source) Type() config.SourceType {
	return s.config.Type
}

func (s *Source) URL() string {
	return s.config.URL
}

func (s *Source) Name() string {
	return s.config.Name
}

func (s *Source) Priority() int {
	return s.config.Priority
}

func (s *Source) Config() config.Source {
	return s.config
}

func (s *Source) GetSource(_ string) ([]byte, error) {
	return s.data, nil
}
func NewSourceManager(sources []config.Source, logger *slog.Logger) *SourceManager {
	cfg := &config.Config{Sources: sources}
	cfg.Fixture()
	return NewSourceManagerWithConfig(cfg, logger)
}

func NewSourceManagerWithConfig(cfg *config.Config, logger *slog.Logger) *SourceManager {
	var sources []config.Source
	if cfg != nil {
		sources = cfg.Sources
	}

	sm := &SourceManager{
		cfg:     cfg,
		sources: make(map[string]*Source),
		ticker:  time.NewTicker(1 * time.Minute), // 每分钟检查一次
		done:    make(chan bool),
		refresh: make(chan bool),
	}

	if logger != nil {
		sm.logger = logger.With("manager", "source_manager")
	}

	for _, s := range sources {
		src := &Source{
			config: s,
		}
		// 如果开启了磁盘持久化，尝试预加载本地快照
		if cfg != nil && cfg.PersistenceOpt.Enable {
			diskPath := sm.getDiskSnapshotPath(s.Name)
			if diskData, err := os.ReadFile(diskPath); err == nil && len(diskData) > 0 {
				src.data = diskData
				src.lastUpdate = time.Now()
				src.version = 1
			}
		}
		sm.sources[s.Name] = src
	}

	go sm.refreshLoop()

	return sm
}

func (sm *SourceManager) getDiskSnapshotPath(name string) string {
	if sm.cfg == nil {
		return ""
	}
	dir := sm.cfg.PersistenceOpt.Dir
	if dir == "" {
		dir = "./data/cache"
	}
	_ = os.MkdirAll(dir, 0755)
	safeName := regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(name, "_")
	return filepath.Join(dir, safeName+".bak")
}

func (sm *SourceManager) log(format string, args ...any) {
	if sm.logger != nil {
		sm.logger.Info(fmt.Sprintf(format, args...))
	}
}

func (sm *SourceManager) refreshLoop() {
	for {
		select {
		case <-sm.ticker.C:
			sm.refreshExpiredSources(false)
		case force := <-sm.refresh:
			sm.refreshExpiredSources(force)
		case <-sm.done:
			sm.ticker.Stop()
			return
		}
	}
}

func (sm *SourceManager) refreshExpiredSources(force bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for name, source := range sm.sources {
		if force ||
			(source.config.Interval != -1 &&
				time.Since(source.lastUpdate) > time.Duration(source.config.Interval)*time.Second) {
			go sm.refreshSource(name) // 异步刷新，避免阻塞
		}
	}
}

func (sm *SourceManager) GetSource(name string) (*Source, error) {
	sm.mu.RLock()
	source, ok := sm.sources[name]
	sm.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("source not found: %s", name)
	}

	if source.config.Interval != -1 &&
		(time.Since(source.lastUpdate) > time.Duration(source.config.Interval)*time.Second || source.data == nil) {
		if err := sm.refreshSource(name); err != nil {
			return nil, err
		}
	}

	return source, nil
}

func (sm *SourceManager) refreshSource(name string) error {
	sm.mu.Lock()
	source, ok := sm.sources[name]
	if !ok {
		sm.mu.Unlock()
		return fmt.Errorf("source not found: %s", name)
	}

	// 如果已经在刷新中，等待正在进行的刷新完成，而不是直接返回空数据
	if source.refreshing {
		ch := source.refreshCh
		sm.mu.Unlock()
		if ch != nil {
			<-ch
		}
		return nil
	}

	// 指数退避
	if !source.lastError.IsZero() {
		backoff := time.Duration(math.Pow(2, float64(source.errorCount))) * time.Second
		if time.Since(source.lastError) < backoff {
			sm.mu.Unlock()
			return fmt.Errorf("too many errors, try again later")
		}
	}

	source.refreshing = true
	source.refreshCh = make(chan struct{})
	sm.mu.Unlock()

	defer func() {
		sm.mu.Lock()
		source.refreshing = false
		if source.refreshCh != nil {
			close(source.refreshCh)
			source.refreshCh = nil
		}
		sm.mu.Unlock()
	}()

	var data []byte
	var err error

	defer func() {
		sm.log("refresh source %s: %v", name, err)
	}()
	timeout := 15
	if sm.cfg != nil && sm.cfg.NetworkOpt.Timeout > 0 {
		timeout = sm.cfg.NetworkOpt.Timeout
	}
	if source.config.Timeout > 0 {
		timeout = source.config.Timeout
	}

	retry := 0
	if sm.cfg != nil && sm.cfg.NetworkOpt.Retry > 0 {
		retry = sm.cfg.NetworkOpt.Retry
	}

	proxy := ""
	if sm.cfg != nil && sm.cfg.NetworkOpt.ProxyURL != "" {
		proxy = sm.cfg.NetworkOpt.ProxyURL
	}
	if source.config.ProxyURL != "" {
		proxy = source.config.ProxyURL
	}

	headers := make(map[string]string)
	if sm.cfg != nil {
		for k, v := range sm.cfg.NetworkOpt.Headers {
			headers[k] = v
		}
	}
	for k, v := range source.config.Headers {
		headers[k] = v
	}

	fetchOpts := config.FetchOptions{
		Timeout:  time.Duration(timeout) * time.Second,
		Retry:    retry,
		ProxyURL: proxy,
		Headers:  headers,
	}

	switch source.Type() {
	case config.SourceTypeTvBoxSingle, config.SourceTypeTvBoxMulti:
		data, err = config.LoadTvBoxDataWithOptions(source.config.URL, fetchOpts)
	case config.SourceTypeEPG, config.SourceTypeM3U, config.SourceTypeTXT, config.SourceTypeDIYP:
		data, err = config.FetchDataWithOptions(source.config.URL, fetchOpts)
	}

	source.mu.Lock()
	defer source.mu.Unlock()

	diskPath := sm.getDiskSnapshotPath(name)

	if err != nil {
		// 故障降级逻辑 (Stale-On-Error)
		if sm.cfg != nil && sm.cfg.PersistenceOpt.Enable && sm.cfg.PersistenceOpt.StaleOnError {
			if source.data != nil {
				sm.log("source %s fetch failed, retaining existing in-memory snapshot: %v", name, err)
				return nil
			}
			if diskPath != "" {
				if diskData, readErr := os.ReadFile(diskPath); readErr == nil && len(diskData) > 0 {
					source.data = diskData
					source.lastUpdate = time.Now()
					source.version++
					sm.log("source %s fetch failed, fallback to disk snapshot (%s): %v", name, diskPath, err)
					return nil
				}
			}
		}

		source.lastError = time.Now()
		source.errorCount++
		return err
	}

	source.data = data
	source.lastUpdate = time.Now()
	source.lastError = time.Time{}
	source.errorCount = 0
	source.version++

	if diskPath != "" && sm.cfg != nil && sm.cfg.PersistenceOpt.Enable {
		_ = os.WriteFile(diskPath, data, 0644)
	}
	return nil
}

func (sm *SourceManager) GetFingerprint(names ...string) string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var b strings.Builder
	if len(names) == 0 {
		keys := make([]string, 0, len(sm.sources))
		for k := range sm.sources {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, name := range keys {
			s := sm.sources[name]
			s.mu.RLock()
			b.WriteString(fmt.Sprintf("%s:%d:%d;", name, s.version, s.lastUpdate.UnixNano()))
			s.mu.RUnlock()
		}
		return b.String()
	}

	for _, name := range names {
		if s, ok := sm.sources[name]; ok {
			s.mu.RLock()
			b.WriteString(fmt.Sprintf("%s:%d:%d;", name, s.version, s.lastUpdate.UnixNano()))
			s.mu.RUnlock()
		}
	}
	return b.String()
}

func (sm *SourceManager) Close() {
	sm.done <- true
}

func (sm *SourceManager) TriggerRefresh(force bool) {
	sm.refresh <- force
}
