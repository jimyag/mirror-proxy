package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"

	_ "github.com/jimmicro/version"

	"github.com/jimyag/mirror-proxy/config"
	"github.com/jimyag/mirror-proxy/execute"
)

var configFilePath string

func init() {
	flag.StringVar(&configFilePath, "f", "config.yaml", "config file path")
	flag.Parse()
}

func main() {
	cfg := config.Config{}
	err := cfg.Load(configFilePath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	executer, err := execute.NewExecuter(cfg)
	if err != nil {
		slog.Error("failed to create executer", "error", err)
		os.Exit(1)
	}

	http.HandleFunc("/", executer.Handle)

	slog.Info("listening on", "listen", cfg.Listen)
	if err := http.ListenAndServe(cfg.Listen, nil); err != nil {
		slog.Error("failed to listen and serve", "error", err)
		os.Exit(1)
	}
}
