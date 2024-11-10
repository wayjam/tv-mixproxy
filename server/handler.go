package server

import (
	"compress/gzip"
	"encoding/xml"
	"image/png"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"

	"github.com/wayjam/tv-mixproxy/config"
	"github.com/wayjam/tv-mixproxy/pkg/imageutil"
	"github.com/wayjam/tv-mixproxy/pkg/m3u"
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
	return png.Encode(c, img)
}

func NewRepoHandler(cfg *config.Config, sourceManager *mixer.SourceManager) fiber.Handler {
	return func(c fiber.Ctx) error {
		if cfg.TvBoxSingleRepoOpt.Disable {
			return c.Status(fiber.StatusNotImplemented).SendString("SingleRepo is disabled")
		}

		result, err := mixer.MixTvBoxRepo(cfg, sourceManager)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		return c.JSON(result)
	}
}

func NewMultiRepoHandler(cfg *config.Config, sourceManager *mixer.SourceManager) fiber.Handler {
	return func(c fiber.Ctx) error {
		if cfg.TvBoxMultiRepoOpt.Disable {
			return c.Status(fiber.StatusNotImplemented).SendString("MultiRepo is disabled")
		}

		result, err := mixer.MixMultiRepo(cfg, sourceManager)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		return c.JSON(result)
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

func NewEPGHandler(cfg *config.Config, sourceManager *mixer.SourceManager) fiber.Handler {
	return func(c fiber.Ctx) error {
		if cfg.EPGOpt.Disable {
			return c.Status(fiber.StatusNotImplemented).SendString("EPG is disabled")
		}

		result, err := mixer.MixEPG(cfg, sourceManager)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		format := c.Query("format", "xml")

		c.Set("Content-Type", "application/xml")
		c.Set("Content-Encoding", "gzip")

		if format == "gz" {
			c.Set("Content-Disposition", "attachment; filename=epg.xml.gz")
		}

		gzipWriter := gzip.NewWriter(c)
		defer gzipWriter.Close()

		encoder := xml.NewEncoder(gzipWriter)
		if err := encoder.Encode(result); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Error encoding and compressing XML")
		}

		return nil
	}
}

func NewM3UMediaHandler(cfg *config.Config, sourceManager *mixer.SourceManager) fiber.Handler {
	return func(c fiber.Ctx) error {
		if cfg.M3UOpt.Disable {
			return c.Status(fiber.StatusNotImplemented).SendString("M3U is disabled")
		}

		result, err := mixer.MixM3UMediaPlayList(cfg, sourceManager)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		encoder := m3u.NewEncoder(c)
		if err := encoder.Encode(result); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		return nil
	}
}

func RefershSrouceHandler(cfg *config.Config, sourceManager *mixer.SourceManager) fiber.Handler {
	return func(c fiber.Ctx) error {
		cronSecret := os.Getenv("CRON_SECRET")
		token := os.Getenv("TV_MIXPROXY_SECRET")

		pass := false

		if cronSecret != "" && c.Get("Authorization") == "Bearer "+cronSecret {
			pass = true
		} else if token != "" && c.Get("X-TV-MIXPROXY-SECRET") == token {
			pass = true
		}

		if !pass {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		go sourceManager.TriggerRefresh(false)
		return c.SendStatus(fiber.StatusOK)
	}
}
