package mixer

import (
	"fmt"
	"math"
	"sync"
	"time"

	fiberlog "github.com/gofiber/fiber/v3/log"

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
	logger  fiberlog.CommonLogger
}

type Source struct {
	config     config.Source
	lastUpdate time.Time
	data       []byte // Change this to []byte
	lastError  time.Time
	errorCount int
}

func (s *Source) Data() []byte {
	return s.data
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

func NewSourceManager(sources []config.Source, logger fiberlog.CommonLogger) *SourceManager {
	sm := &SourceManager{
		sources: make(map[string]*Source),
		ticker:  time.NewTicker(1 * time.Minute), // 每分钟检查一次
		done:    make(chan bool),
		refresh: make(chan bool),
		logger:  logger,
	}

	for _, s := range sources {
		sm.sources[s.Name] = &Source{
			config: s,
		}
	}

	go sm.refreshLoop()

	return sm
}

func (sm *SourceManager) refreshLoop() {
	for {
		select {
		case <-sm.ticker.C:
			sm.refreshExpiredSources()
		case <-sm.refresh:
			sm.refreshExpiredSources()
		case <-sm.done:
			sm.ticker.Stop()
			return
		}
	}
}

func (sm *SourceManager) refreshExpiredSources() {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for name, source := range sm.sources {
		if time.Since(source.lastUpdate) > time.Duration(source.config.Interval)*time.Second {
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

	if time.Since(source.lastUpdate) > time.Duration(source.config.Interval)*time.Second || source.data == nil {
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

	// 指数退避
	if !source.lastError.IsZero() {
		backoff := time.Duration(math.Pow(2, float64(source.errorCount))) * time.Second
		if time.Since(source.lastError) < backoff {
			sm.mu.Unlock()
			return fmt.Errorf("too many errors, try again later")
		}
	}

	sm.mu.Unlock()

	var data []byte
	var err error

	defer func() {
		if sm.logger != nil {
			sm.logger.Infow("refresh source", "name", name, "error", err)
		}
	}()

	switch source.Type() {
	case config.SourceTypeTvBoxSingle:
		data, err = config.LoadTvBoxData(source.config.URL)
	case config.SourceTypeTvBoxMulti:
		data, err = config.LoadTvBoxData(source.config.URL)
	case config.SourceTypeEPG:
		data, err = config.FetchData(source.config.URL)
	case config.SourceTypeM3U:
		data, err = config.FetchData(source.config.URL)
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	if err != nil {
		source.lastError = time.Now()
		source.errorCount++
		return err
	}

	source.data = data
	source.lastUpdate = time.Now()
	source.lastError = time.Time{}
	source.errorCount = 0
	return nil
}

func (sm *SourceManager) Close() {
	sm.done <- true
}

func (sm *SourceManager) TriggerRefresh() {
	sm.refresh <- true
}
