// Package api 提供 Serverless HTTP API 服务
package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/spf13/viper"

	"github.com/wayjam/tv-mixproxy/config"
	"github.com/wayjam/tv-mixproxy/server"
)

var (
	app http.Handler
)

// Entrypoint
func Handler(w http.ResponseWriter, r *http.Request) {
	app.ServeHTTP(w, r)
}

func init() {
	// load config from remote
	cfg, err := loadRemoteConfig()
	if err != nil {
		panic(err)
	}

	server := server.NewServer(cfg)
	if err := server.PreRun(); err != nil {
		panic(err)
	}
	app = adaptor.FiberApp(server.App())
}

func loadRemoteConfig() (*config.Config, error) {
	configURL := os.Getenv("TV_MIXPROXY_CFG_URL")
	if configURL == "" {
		slog.Info("no config url provided, using default config")
		return config.DefaultConfig(), nil
	}

	resp, err := http.Get(configURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get config from remote %s", err)
	}
	defer resp.Body.Close()

	v := viper.New()
	v.SetConfigType("yaml")
	v.ReadConfig(resp.Body)

	return config.UnmarshalConfig(v)
}
