package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"xrp/internal/config"
	"xrp/internal/proxy"
)

var version string = "<dev>"

// parseLogLevel converts a string log level to slog.Level
func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		slog.Warn("Invalid log level, defaulting to info", "level", level)
		return slog.LevelInfo
	}
}

func main() {
	var configFile string
	var addr string
	var logLevel string

	flag.StringVar(&configFile, "config", "config.json", "Path to configuration file")
	flag.StringVar(&addr, "addr", ":8080", "Address to listen on")
	flag.StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(logLevel),
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load(configFile)
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	proxyServer, err := proxy.New(cfg, version)
	if err != nil {
		slog.Error("Failed to create proxy server", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:    addr,
		Handler: proxyServer,
	}

	go func() {
		slog.Info("Starting server", "addr", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	for {
		sig := <-sigChan
		switch sig {
		case syscall.SIGHUP:
			slog.Info("Reloading configuration")
			newCfg, err := config.Load(configFile)
			if err != nil {
				slog.Error("Failed to reload configuration", "error", err)
				continue
			}
			if err := proxyServer.UpdateConfig(newCfg); err != nil {
				slog.Error("Failed to update proxy configuration", "error", err)
				continue
			}
			slog.Info("Configuration reloaded successfully")
		case syscall.SIGINT, syscall.SIGTERM:
			slog.Info("Shutting down server")
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := server.Shutdown(ctx); err != nil {
				slog.Error("Server shutdown failed", "error", err)
				os.Exit(1)
			}
			return
		}
	}
}
