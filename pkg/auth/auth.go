package auth

import (
	"crypto/subtle"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/wayjam/tv-mixproxy/config"
)

var (
	ErrMissingToken = errors.New("missing access key or token")
	ErrInvalidToken = errors.New("invalid access key or token")
	ErrUserDisabled = errors.New("user account or access key is disabled")
	ErrTokenExpired = errors.New("access key or token has expired")
)

// User 表示已通过鉴权的用户上下文
type User struct {
	ID            string
	Key           string
	Enabled       bool
	ExpireAt      time.Time
	HasExpire     bool
	AllowedGroups map[string]struct{}
	AllowAll      bool
}

// IsGroupAllowed 检查当前用户是否有权访问指定的频道分组
func (u *User) IsGroupAllowed(groupName string) bool {
	if u == nil || u.AllowAll {
		return true
	}
	if groupName == "" {
		return true
	}
	_, ok := u.AllowedGroups[groupName]
	return ok
}

// Manager 统一安全鉴权管理器
type Manager struct {
	mu        sync.RWMutex
	opt       config.SecurityOpt
	userByKey map[string]*User
	userByID  map[string]*User
}

// NewManager 创建鉴权管理器
func NewManager(opt config.SecurityOpt) *Manager {
	m := &Manager{
		opt:       opt,
		userByKey: make(map[string]*User),
		userByID:  make(map[string]*User),
	}
	m.loadUsers()
	return m
}

func (m *Manager) loadUsers() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.userByKey = make(map[string]*User)
	m.userByID = make(map[string]*User)

	// 1. 注册多用户列表
	for _, u := range m.opt.Users {
		if u.Key == "" {
			continue
		}

		user := &User{
			ID:            u.ID,
			Key:           u.Key,
			Enabled:       u.IsEnabled(),
			AllowedGroups: make(map[string]struct{}),
			AllowAll:      false,
		}

		if user.ID == "" {
			user.ID = u.Key
		}

		// 解析过期时间
		if u.ExpireAt != "" {
			parsedTime, err := parseTime(u.ExpireAt)
			if err == nil {
				user.ExpireAt = parsedTime
				user.HasExpire = true
			}
		}

		// 解析频道分组白名单
		if len(u.AllowedGroups) == 0 {
			user.AllowAll = true
		} else {
			for _, g := range u.AllowedGroups {
				if g == "*" {
					user.AllowAll = true
					break
				}
				user.AllowedGroups[g] = struct{}{}
			}
		}

		m.userByKey[u.Key] = user
		m.userByID[user.ID] = user
	}

	// 2. 注册单用户 AccessKey / Secret
	singleKey := m.opt.GetSecret()
	if singleKey != "" {
		if _, exists := m.userByKey[singleKey]; !exists {
			defaultUser := &User{
				ID:       "default",
				Key:      singleKey,
				Enabled:  true,
				AllowAll: true,
			}
			m.userByKey[singleKey] = defaultUser
			m.userByID["default"] = defaultUser
		}
	}
}

// Authenticate 校验 Token 并返回对应的用户模型
func (m *Manager) Authenticate(token string) (*User, error) {
	if !m.opt.Enable {
		// 未开启鉴权时直接返回默认全权限用户
		return &User{ID: "anonymous", Enabled: true, AllowAll: true}, nil
	}

	if token == "" {
		return nil, ErrMissingToken
	}

	m.mu.RLock()
	user, exists := m.userByKey[token]
	m.mu.RUnlock()

	if !exists {
		return nil, ErrInvalidToken
	}

	// 常量时间比较防止时序侧信道泄露
	if subtle.ConstantTimeCompare([]byte(user.Key), []byte(token)) != 1 {
		return nil, ErrInvalidToken
	}

	if !user.Enabled {
		return nil, ErrUserDisabled
	}

	if user.HasExpire && time.Now().After(user.ExpireAt) {
		return nil, ErrTokenExpired
	}

	return user, nil
}

// IsPathWhitelisted 检查请求路径是否在免鉴权白名单中
func (m *Manager) IsPathWhitelisted(reqPath string) bool {
	cleanPath := strings.TrimSpace(reqPath)
	if cleanPath == "" {
		cleanPath = "/"
	}

	for _, w := range m.opt.WhitelistPaths {
		if w == "" {
			continue
		}
		if w == cleanPath {
			return true
		}
		if w != "/" && strings.HasPrefix(cleanPath, w) {
			return true
		}
	}
	return false
}

// SecurityOpt 返回当前安全配置
func (m *Manager) SecurityOpt() config.SecurityOpt {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.opt
}

func parseTime(str string) (time.Time, error) {
	str = strings.TrimSpace(str)
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02",
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.ParseInLocation(f, str, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("unsupported time format")
}
