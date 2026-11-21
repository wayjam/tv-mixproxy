package server

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/wayjam/tv-mixproxy/pkg/auth"
)

const (
	ContextUserKey  = "auth_user"
	ContextTokenKey = "auth_token"
)

// NewAuthMiddleware 创建统一安全鉴权中间件
func NewAuthMiddleware(authMgr *auth.Manager) fiber.Handler {
	return func(c fiber.Ctx) error {
		if authMgr == nil {
			return c.Next()
		}

		secOpt := authMgr.SecurityOpt()
		token := extractToken(c, secOpt.TokenParam, secOpt.HeaderName)

		// 检查路径白名单
		reqPath := c.Path()
		isWhitelisted := authMgr.IsPathWhitelisted(reqPath)

		// 如果未开启全局鉴权
		if !secOpt.Enable {
			return c.Next()
		}

		user, err := authMgr.Authenticate(token)
		if err != nil {
			// 如果在白名单内，即使鉴权失败也允许匿名访问（如 /logo, /wallpaper）
			if isWhitelisted {
				return c.Next()
			}
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized: " + err.Error())
		}

		// 挂载鉴权上下文
		c.Locals(ContextUserKey, user)
		c.Locals(ContextTokenKey, token)

		return c.Next()
	}
}

// extractToken 从多种来源优先提取 Token：Query 参数 -> 自定义 Header -> Authorization Bearer
func extractToken(c fiber.Ctx, tokenParam, headerName string) string {
	// 1. Query 参数（如 ?token=xxx 或 ?key=xxx）
	if tokenParam == "" {
		tokenParam = "token"
	}
	if q := c.Query(tokenParam); q != "" {
		return strings.TrimSpace(q)
	}
	if q := c.Query("key"); q != "" {
		return strings.TrimSpace(q)
	}
	if q := c.Query("access_key"); q != "" {
		return strings.TrimSpace(q)
	}

	// 2. 自定义 Header（如 X-Access-Key）
	if headerName == "" {
		headerName = "X-Access-Key"
	}
	if h := c.Get(headerName); h != "" {
		return strings.TrimSpace(h)
	}

	// 3. Authorization: Bearer <token>
	if authHeader := c.Get("Authorization"); authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}

	// 4. 兼容 X-TV-MIXPROXY-SECRET
	if secretHeader := c.Get("X-TV-MIXPROXY-SECRET"); secretHeader != "" {
		return strings.TrimSpace(secretHeader)
	}

	return ""
}

// GetCurrentUser 从上下文获取当前鉴权用户
func GetCurrentUser(c fiber.Ctx) *auth.User {
	if val := c.Locals(ContextUserKey); val != nil {
		if u, ok := val.(*auth.User); ok {
			return u
		}
	}
	return nil
}

// GetCurrentToken 从上下文获取当前使用的 Token
func GetCurrentToken(c fiber.Ctx) string {
	if val := c.Locals(ContextTokenKey); val != nil {
		if t, ok := val.(string); ok {
			return t
		}
	}
	return ""
}
