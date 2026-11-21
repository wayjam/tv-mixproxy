package server

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"image/png"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"

	"github.com/wayjam/tv-mixproxy/config"
	"github.com/wayjam/tv-mixproxy/pkg/cache"
	"github.com/wayjam/tv-mixproxy/pkg/imageutil"
	"github.com/wayjam/tv-mixproxy/pkg/mixer"
)

func Home(c fiber.Ctx) error {
	return c.SendString("Hello, TV MixProxy 📺!")
}

func generateLogoSVG() string {
	return `<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 100 100">
		<!-- TV Body with rounded corners and darker green -->
		<rect x="10" y="15" width="80" height="60" rx="15" ry="15" fill="#2E8B57"/>
		
		<!-- TV Screen with white -->
		<rect x="15" y="20" width="70" height="45" rx="10" ry="10" fill="#FFFFFF"/>
		
		<!-- Kawaii face on screen -->
		<!-- Sparkly eyes -->
		<circle cx="35" cy="40" r="6" fill="#006400"/>
		<circle cx="65" cy="40" r="6" fill="#006400"/>
		<circle cx="33" cy="38" r="2" fill="#FFFFFF"/>
		<circle cx="63" cy="38" r="2" fill="#FFFFFF"/>
		
		<!-- Rosy cheeks -->
		<circle cx="30" cy="45" r="4" fill="#FF69B4" opacity="0.7"/>
		<circle cx="70" cy="45" r="4" fill="#FF69B4" opacity="0.7"/>
		
		<!-- Cute smile -->
		<path d="M40 48 Q50 55 60 48" stroke="#006400" stroke-width="3" fill="none"/>
		
		<!-- Bouncy antenna with leaves -->
		<path d="M30 15 Q40 0 50 10" stroke="#228B22" stroke-width="3" fill="none"/>
		<path d="M70 15 Q60 0 50 10" stroke="#228B22" stroke-width="3" fill="none"/>
		
		<!-- Little leaves -->
		<path d="M50 10 Q53 8 50 6 Q47 8 50 10" fill="#006400"/>
		<path d="M30 15 Q33 13 30 11 Q27 13 30 15" fill="#006400"/>
		<path d="M70 15 Q73 13 70 11 Q67 13 70 15" fill="#006400"/>
		
		<!-- Decorative buttons -->
		<circle cx="25" cy="70" r="3" fill="#006400"/>
		<circle cx="75" cy="70" r="3" fill="#006400"/>
	</svg>`
}

func Logo(c fiber.Ctx) error {
	c.Set("Content-Type", "image/svg+xml")
	c.Set("Cache-Control", "public, max-age=86400")
	return c.SendString(generateLogoSVG())
}

func Wallpaper(c fiber.Ctx) error {
	// Parse query parameters
	width, _ := strconv.Atoi(c.Query("width", "800"))
	height, _ := strconv.Atoi(c.Query("height", "600"))
	pattern := c.Query("pattern", "solid")
	opacity, _ := strconv.ParseFloat(c.Query("opacity", "1.0"), 64)
	borderWidth, _ := strconv.Atoi(c.Query("border_width", "0"))

	// Parse colors
	bgColor := imageutil.ParseColor(c.Query("bg_color", "FFFFFF"))
	borderColor := imageutil.ParseColor(c.Query("border_color", "000000"))

	params := imageutil.ImageParams{
		BackgroundColor: bgColor,
		Width:           width,
		Height:          height,
		Pattern:         pattern,
		Opacity:         opacity,
		BorderWidth:     borderWidth,
		BorderColor:     borderColor,
	}

	img := imageutil.GenerateImage(params)
	c.Set("Content-Type", "image/png")
	c.Set("Cache-Control", "public, max-age=86400")
	return png.Encode(c, img)
}

func sendCachedOrCompute(
	c fiber.Ctx,
	cfg *config.Config,
	cacheManager *cache.Manager,
	cacheKey string,
	version string,
	contentType string,
	pinned bool,
	extraHeaders map[string]string,
	compute func() ([]byte, error),
) error {
	maxAge := 60
	if cfg != nil && cfg.CacheOpt.MaxAge > 0 {
		maxAge = cfg.CacheOpt.MaxAge
	}
	cacheControlVal := fmt.Sprintf("public, max-age=%d", maxAge)
	if cfg != nil && cfg.CacheOpt.Disable {
		cacheControlVal = "no-cache, no-store, must-revalidate"
	}

	if cacheManager != nil && (cfg == nil || !cfg.CacheOpt.Disable) {
		if entry, ok := cacheManager.Get(cacheKey); ok && entry.Version == version {
			for k, v := range entry.Header {
				c.Set(k, v)
			}
			c.Set("ETag", entry.ETag)
			c.Set("Cache-Control", cacheControlVal)

			if match := c.Get("If-None-Match"); match != "" && (match == entry.ETag || match == "*") {
				return c.SendStatus(fiber.StatusNotModified)
			}

			c.Set("Content-Type", entry.ContentType)
			return c.Send(entry.Data)
		}
	}

	data, err := compute()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	var etag string
	if cacheManager != nil && (cfg == nil || !cfg.CacheOpt.Disable) {
		var entry cache.Entry
		if pinned {
			entry = cacheManager.SetPinned(cacheKey, data, contentType, version, extraHeaders)
		} else {
			entry = cacheManager.Set(cacheKey, data, contentType, version, extraHeaders)
		}
		etag = entry.ETag
	}

	for k, v := range extraHeaders {
		c.Set(k, v)
	}
	c.Set("Cache-Control", cacheControlVal)
	if etag != "" {
		c.Set("ETag", etag)
		if match := c.Get("If-None-Match"); match != "" && (match == etag || match == "*") {
			return c.SendStatus(fiber.StatusNotModified)
		}
	}

	c.Set("Content-Type", contentType)
	return c.Send(data)
}

func NewRepoHandler(cfg *config.Config, sourceManager *mixer.SourceManager, cacheManager *cache.Manager) fiber.Handler {
	return func(c fiber.Ctx) error {
		if cfg.TvBoxSingleRepoOpt.Disable {
			return c.Status(fiber.StatusNotImplemented).SendString("SingleRepo is disabled")
		}

		userID := getUserID(c)
		version := sourceManager.GetFingerprint()
		cacheKey := fmt.Sprintf("tvbox_repo_%s", userID)
		return sendCachedOrCompute(c, cfg, cacheManager, cacheKey, version, "application/json; charset=utf-8", true, nil, func() ([]byte, error) {
			result, err := mixer.MixTvBoxRepo(cfg, sourceManager, mixer.MixOptions{
				User:  GetCurrentUser(c),
				Token: GetCurrentToken(c),
			})
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		})
	}
}
func NewMultiRepoHandler(cfg *config.Config, sourceManager *mixer.SourceManager, cacheManager *cache.Manager) fiber.Handler {
	return func(c fiber.Ctx) error {
		if cfg.TvBoxMultiRepoOpt.Disable {
			return c.Status(fiber.StatusNotImplemented).SendString("MultiRepo is disabled")
		}

		userID := getUserID(c)
		version := sourceManager.GetFingerprint()
		cacheKey := fmt.Sprintf("tvbox_multi_repo_%s", userID)
		return sendCachedOrCompute(c, cfg, cacheManager, cacheKey, version, "application/json; charset=utf-8", true, nil, func() ([]byte, error) {
			result, err := mixer.MixMultiRepo(cfg, sourceManager, mixer.MixOptions{
				User:  GetCurrentUser(c),
				Token: GetCurrentToken(c),
			})
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		})
	}
}
func NewSpiderHandler(cfg *config.Config, sourceManager *mixer.SourceManager) fiber.Handler {
	handler, err := mixer.NewMixURLHandler(cfg.TvBoxSingleRepoOpt.Spider, sourceManager)

	return func(c fiber.Ctx) error {
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		// Convert fiber.Ctx to http.ResponseWriter and *http.Request
		return adaptor.HTTPHandler(handler)(c)
	}
}

func NewEPGHandler(cfg *config.Config, sourceManager *mixer.SourceManager, cacheManager *cache.Manager) fiber.Handler {
	return func(c fiber.Ctx) error {
		if cfg.EPGOpt.Disable {
			return c.Status(fiber.StatusNotImplemented).SendString("EPG is disabled")
		}

		format := c.Query("format", "xml")
		userID := getUserID(c)
		version := sourceManager.GetFingerprint()

		if format == "gz" {
			headers := map[string]string{
				"Content-Disposition": "attachment; filename=epg.xml.gz",
			}
			cacheKey := fmt.Sprintf("epg_xml_gz_%s", userID)
			return sendCachedOrCompute(c, cfg, cacheManager, cacheKey, version, "application/gzip", true, headers, func() ([]byte, error) {
				result, err := mixer.MixEPG(cfg, sourceManager)
				if err != nil {
					return nil, err
				}
				buf := new(bytes.Buffer)
				gzWriter := gzip.NewWriter(buf)
				if _, err := gzWriter.Write([]byte(xml.Header)); err != nil {
					return nil, err
				}
				encoder := xml.NewEncoder(gzWriter)
				encoder.Indent("", "  ")
				if err := encoder.Encode(result); err != nil {
					return nil, err
				}
				if err := gzWriter.Close(); err != nil {
					return nil, err
				}
				return buf.Bytes(), nil
			})
		}

		cacheKey := fmt.Sprintf("epg_xml_%s", userID)
		return sendCachedOrCompute(c, cfg, cacheManager, cacheKey, version, "application/xml; charset=utf-8", true, nil, func() ([]byte, error) {
			result, err := mixer.MixEPG(cfg, sourceManager)
			if err != nil {
				return nil, err
			}
			buf := new(bytes.Buffer)
			if _, err := buf.Write([]byte(xml.Header)); err != nil {
				return nil, err
			}
			encoder := xml.NewEncoder(buf)
			encoder.Indent("", "  ")
			if err := encoder.Encode(result); err != nil {
				return nil, err
			}
			return buf.Bytes(), nil
		})
	}
}

func NewM3UMediaHandler(cfg *config.Config, sourceManager *mixer.SourceManager, cacheManager *cache.Manager) fiber.Handler {
	return func(c fiber.Ctx) error {
		if cfg.M3UOpt.Disable {
			return c.Status(fiber.StatusNotImplemented).SendString("M3U is disabled")
		}

		userID := getUserID(c)
		version := sourceManager.GetFingerprint()
		cacheKey := fmt.Sprintf("m3u_media_playlist_%s", userID)
		return sendCachedOrCompute(c, cfg, cacheManager, cacheKey, version, "application/vnd.apple.mpegurl; charset=utf-8", true, nil, func() ([]byte, error) {
			result, err := mixer.MixM3UMediaPlayList(cfg, sourceManager, mixer.MixOptions{
				User:  GetCurrentUser(c),
				Token: GetCurrentToken(c),
			})
			if err != nil {
				return nil, err
			}
			return result.MarshalM3U()
		})
	}
}

func RefershSrouceHandler(cfg *config.Config, sourceManager *mixer.SourceManager, cacheManager *cache.Manager) fiber.Handler {
	return func(c fiber.Ctx) error {
		cronSecret := os.Getenv("CRON_SECRET")
		token := os.Getenv("TV_MIXPROXY_SECRET")
		if cfg != nil && cfg.SecurityOpt.GetSecret() != "" {
			token = cfg.SecurityOpt.GetSecret()
		}
		pass := false
		authHeader := c.Get("Authorization")
		secretHeader := c.Get("X-TV-MIXPROXY-SECRET")
		queryToken := c.Query("token", c.Query("secret"))

		if cronSecret != "" && authHeader == "Bearer "+cronSecret {
			pass = true
		} else if token != "" && (secretHeader == token || authHeader == "Bearer "+token || queryToken == token) {
			pass = true
		}

		if !pass {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		go func() {
			sourceManager.TriggerRefresh(false)
			if cacheManager != nil {
				cacheManager.Clear()
			}
		}()
		return c.SendStatus(fiber.StatusOK)
	}
}

func NewLiveTXTHandler(cfg *config.Config, sourceManager *mixer.SourceManager, cacheManager *cache.Manager) fiber.Handler {
	return func(c fiber.Ctx) error {
		if cfg.M3UOpt.Disable {
			return c.Status(fiber.StatusNotImplemented).SendString("Live list is disabled")
		}

		userID := getUserID(c)
		version := sourceManager.GetFingerprint()
		cacheKey := fmt.Sprintf("live_txt_%s", userID)
		return sendCachedOrCompute(c, cfg, cacheManager, cacheKey, version, "text/plain; charset=utf-8", true, nil, func() ([]byte, error) {
			result, err := mixer.MixTXT(cfg, sourceManager, mixer.MixOptions{
				User:  GetCurrentUser(c),
				Token: GetCurrentToken(c),
			})
			if err != nil {
				return nil, err
			}
			return result.Marshal()
		})
	}
}

func NewDIYPHandler(cfg *config.Config, sourceManager *mixer.SourceManager, cacheManager *cache.Manager) fiber.Handler {
	return func(c fiber.Ctx) error {
		if cfg.EPGOpt.Disable {
			return c.Status(fiber.StatusNotImplemented).SendString("EPG is disabled")
		}

		ch := c.Query("ch")
		date := c.Query("date")
		version := sourceManager.GetFingerprint()
		userID := getUserID(c)
		cacheKey := fmt.Sprintf("diyp_%s_%s_%s", ch, date, userID)
		return sendCachedOrCompute(c, cfg, cacheManager, cacheKey, version, "application/json; charset=utf-8", false, nil, func() ([]byte, error) {
			result, err := mixer.QueryDIYPEPG(cfg, sourceManager, ch, date)
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		})
	}
}

func getUserID(c fiber.Ctx) string {
	if u := GetCurrentUser(c); u != nil && u.ID != "" {
		return u.ID
	}
	return "public"
}
