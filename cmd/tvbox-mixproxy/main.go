package main

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/wayjam/tvbox-mixproxy/config"
	"github.com/wayjam/tvbox-mixproxy/server"
)

func main() {
	var cfgFile string
	var port int
	rootCmd := &cobra.Command{
		Use:   "tvbox-mixproxy",
		Short: "TVBox MixProxy server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadServerConfig(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if port != 0 {
				cfg.ServerPort = port
			}

			svr := server.NewServer(cfg)
			return svr.Run()
		},
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.tvbox_mixproxy.yaml)")
	rootCmd.PersistentFlags().IntVar(&port, "port", 8080, "server port (overrides config file if specified)")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
