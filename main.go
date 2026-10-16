package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"relay/config"
	"relay/connection"
	"relay/logging"
	"relay/plugin"
	"time"
)

const defaultPluginDirectory = "../workspace/plugins"

var running = true

func main() {
	logging.Setup()
	plugin.Init()

	pluginDirectory := os.Getenv("RELAY_PLUGIN_DIR")
	if pluginDirectory == "" {
		pluginDirectory = defaultPluginDirectory
	}

	if err := plugin.EnsureDirectory(pluginDirectory); err != nil {
		slog.Error("ensure plugin directory", "err", err)
		os.Exit(1)
	}

	if _, err := plugin.LoadDirectory(context.Background(), pluginDirectory, plugin.DefaultBus); err != nil {
		slog.Error("load plugins", "err", err)
		os.Exit(1)
	}

	startTime := time.Now()
	cfg, err := config.LoadConfig("config.yml")
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	slog.Info("loaded config")

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Network.Port))
	if err != nil {
		slog.Error("listen", "err", err)
		os.Exit(1)
	}
	defer listener.Close()

	ms := float32(time.Since(startTime).Microseconds()) / 1000.0
	slog.Info("started", "ms", fmt.Sprintf("%.2f", ms), "port", cfg.Network.Port)

	for running {
		conn, err := listener.Accept()
		if err != nil {
			slog.Error("accept", "err", err)
			continue
		}

		go connection.HandleConnection(conn, cfg, plugin.DefaultBus)
	}
}

func IsServerRunning() bool {
	return running
}

func StopProxy() {
	running = false
}
