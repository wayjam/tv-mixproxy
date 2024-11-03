package server

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/gofiber/fiber/v3"
	fiberlog "github.com/gofiber/fiber/v3/log"
	"github.com/gofiber/fiber/v3/middleware/logger"
	recoverer "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/wayjam/tv-mixproxy/config"
	"github.com/wayjam/tv-mixproxy/pkg/mixer"
)

type server struct {
	app           *fiber.App
	cfg           *config.Config
	sourceManager *mixer.SourceManager
}

func NewServer(cfg *config.Config) *server {
	app := fiber.New(fiber.Config{
		AppName: "TV MixProxy",
	})

	app.Use(recoverer.New(recoverer.Config{
		EnableStackTrace: true,
	}))
	app.Use(requestid.New())

	// Configure logging middleware
	var logOutput io.Writer
	if cfg.Log.Output == "stdout" || cfg.Log.Output == "" {
		logOutput = os.Stdout
	} else {
		logOutput = &lumberjack.Logger{
			Filename:   cfg.Log.Output,
			MaxSize:    100, // megabytes
			MaxBackups: 3,
			MaxAge:     28, // days
		}
	}

	// Set up custom logger format
	fiberlog.SetOutput(logOutput)
	fiberlog.SetLevel(fiberlog.Level(cfg.Log.Level))
	slog.SetDefault(slog.New(slog.NewTextHandler(logOutput, nil)))

	app.Use(logger.New(logger.Config{
		Output:     logOutput,
		Format:     "${time} ${locals:requestid} [${level}] ${status} - ${latency} ${method} ${path}\n",
		TimeFormat: "2006/01/02 15:04:05.000000",
		TimeZone:   "Local",
	}))

	sourceManager := mixer.NewSourceManager(cfg.Sources, slog.Default())

	return &server{
		app:           app,
		cfg:           cfg,
		sourceManager: sourceManager,
	}
}

func (s *server) SetupRoutes() {
	app := s.app
	app.Get("/", Home)
	app.Get("/logo", Logo)
	app.Get("/wallpaper", Wallpaper)

	v1 := app.Group("/v1")
	v1.Get("/tvbox/repo", NewRepoHandler(s.cfg, s.sourceManager))
	v1.Get("/tvbox/multi_repo", NewMultiRepoHandler(s.cfg, s.sourceManager))
	v1.Get("/tvbox/spider", NewSpiderHandler(s.cfg, s.sourceManager))
	v1.Get("/epg.xml", NewEPGHandler(s.cfg, s.sourceManager))
	v1.Get("/m3u/media_playlist", NewM3UMediaHandler(s.cfg, s.sourceManager))
}

func (s *server) App() *fiber.App {
	return s.app
}

func (s *server) PreRun() error {
	if !s.cfg.TvBoxSingleRepoOpt.Disable {
		// Try MixRepo
		_, err := mixer.MixTvBoxRepo(s.cfg, s.sourceManager)
		if err != nil {
			return fmt.Errorf("failed to initialize MixRepo: %w", err)
		}
	}

	if !s.cfg.TvBoxMultiRepoOpt.Disable {
		// Try MixMultiRepo
		_, err := mixer.MixMultiRepo(s.cfg, s.sourceManager)
		if err != nil {
			return fmt.Errorf("failed to initialize MixMultiRepo: %w", err)
		}
	}

	s.SetupRoutes()
	s.sourceManager.TriggerRefresh()
	return nil
}

func (s *server) Run() error {
	if err := s.PreRun(); err != nil {
		return err
	}
	return s.app.Listen(fmt.Sprintf(":%d", s.cfg.ServerPort))
}
