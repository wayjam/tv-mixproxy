package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wayjam/tv-mixproxy/config"
)

func TestSingleKeyAuth(t *testing.T) {
	opt := config.SecurityOpt{
		Enable:    true,
		AccessKey: "secret_12345",
	}
	mgr := NewManager(opt)

	// Valid key
	user, err := mgr.Authenticate("secret_12345")
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "default", user.ID)
	assert.True(t, user.IsGroupAllowed("央视频道"))

	// Missing key
	_, err = mgr.Authenticate("")
	assert.ErrorIs(t, err, ErrMissingToken)

	// Invalid key
	_, err = mgr.Authenticate("wrong_key")
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestMultiUserAuthAndExpiration(t *testing.T) {
	f := false
	tr := true
	opt := config.SecurityOpt{
		Enable: true,
		Users: []config.UserOpt{
			{
				ID:            "living_room",
				Key:           "key_living_room",
				Enabled:       &tr,
				AllowedGroups: []string{"*"},
			},
			{
				ID:            "parents",
				Key:           "key_parents",
				Enabled:       &tr,
				AllowedGroups: []string{"央视频道", "卫视频道"},
			},
			{
				ID:      "disabled_user",
				Key:     "key_disabled",
				Enabled: &f,
			},
			{
				ID:       "expired_user",
				Key:      "key_expired",
				Enabled:  &tr,
				ExpireAt: time.Now().Add(-1 * time.Hour).Format("2006-01-02 15:04:05"),
			},
			{
				ID:       "valid_future_user",
				Key:      "key_future",
				Enabled:  &tr,
				ExpireAt: time.Now().Add(24 * time.Hour).Format("2006-01-02"),
			},
		},
	}
	mgr := NewManager(opt)

	// 1. living_room: full access
	u1, err := mgr.Authenticate("key_living_room")
	assert.NoError(t, err)
	assert.Equal(t, "living_room", u1.ID)
	assert.True(t, u1.IsGroupAllowed("任意频道"))

	// 2. parents: restricted groups
	u2, err := mgr.Authenticate("key_parents")
	assert.NoError(t, err)
	assert.Equal(t, "parents", u2.ID)
	assert.True(t, u2.IsGroupAllowed("央视频道"))
	assert.True(t, u2.IsGroupAllowed("卫视频道"))
	assert.False(t, u2.IsGroupAllowed("地方频道"))

	// 3. disabled user -> ErrUserDisabled
	_, err = mgr.Authenticate("key_disabled")
	assert.ErrorIs(t, err, ErrUserDisabled)

	// 4. expired user -> ErrTokenExpired
	_, err = mgr.Authenticate("key_expired")
	assert.ErrorIs(t, err, ErrTokenExpired)

	// 5. valid future user -> success
	u5, err := mgr.Authenticate("key_future")
	assert.NoError(t, err)
	assert.Equal(t, "valid_future_user", u5.ID)
}

func TestWhitelistPaths(t *testing.T) {
	opt := config.SecurityOpt{
		Enable:    true,
		AccessKey: "key123",
		WhitelistPaths: []string{
			"/",
			"/logo",
			"/wallpaper",
		},
	}
	mgr := NewManager(opt)

	assert.True(t, mgr.IsPathWhitelisted("/"))
	assert.True(t, mgr.IsPathWhitelisted("/logo"))
	assert.True(t, mgr.IsPathWhitelisted("/wallpaper"))
	assert.False(t, mgr.IsPathWhitelisted("/v1/m3u/media_playlist"))
	assert.False(t, mgr.IsPathWhitelisted("/v1/live.txt"))
	assert.False(t, mgr.IsPathWhitelisted("/api/diyp"))
}

func TestAuthDisabled(t *testing.T) {
	opt := config.SecurityOpt{
		Enable: false,
	}
	mgr := NewManager(opt)

	user, err := mgr.Authenticate("")
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.True(t, user.IsGroupAllowed("any"))
}
