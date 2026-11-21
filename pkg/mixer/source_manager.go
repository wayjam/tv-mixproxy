package mixer

import (
	"fmt"
	"log/slog"
	"math"
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

func (s *Source) GetSource(_ string) ([]byte, error) {
	return s.data, nil
}

func NewSourceManager(sources []config.Source, logger *slog.Logger) *SourceManager {
	sm := &SourceManager{
		sources: make(map[string]*Source),
		ticker:  time.NewTicker(1 * time.Minute), // 每分钟检查一次
		done:    make(chan bool),
		refresh: make(chan bool),
	}

	if logger != nil {
		sm.logger = logger.With("manager", "source_manager")
	}

	for _, s := range sources {
		sm.sources[s.Name] = &Source{
			config: s,
		}
	}

	go sm.refreshLoop()

	return sm
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

	switch source.Type() {
	case config.SourceTypeTvBoxSingle, config.SourceTypeTvBoxMulti:
		data, err = config.LoadTvBoxData(source.config.URL)
	case config.SourceTypeEPG, config.SourceTypeM3U, config.SourceTypeTXT, config.SourceTypeDIYP:
		data, err = config.FetchData(source.config.URL)
	}

	source.mu.Lock()
	defer source.mu.Unlock()

	if err != nil {
		source.lastError = time.Now()
		source.errorCount++
		return err
	}

	source.data = data
	source.lastUpdate = time.Now()
	source.lastError = time.Time{}
	source.errorCount = 0
	source.version++
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
