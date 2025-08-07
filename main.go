package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"xrp/internal/config"
	"xrp/internal/health"
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
	var showVersion bool
	var logLevel string

	flag.StringVar(&configFile, "config", "config.json", "Path to configuration file")
	flag.StringVar(&addr, "addr", ":8080", "Address to listen on")
	flag.BoolVar(&showVersion, "version", false, "Print version and exit")
	flag.StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	if showVersion {
		println(version)
		os.Exit(0)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(logLevel),
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load(configFile)
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Create health server before proxy to handle startup monitoring
	healthServer := health.New(cfg.HealthPort)
	
	// Start health server in background
	go func() {
		if err := healthServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Health server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Create proxy server (this loads and validates plugins)
	proxyServer, err := proxy.New(cfg, version)
	if err != nil {
		slog.Error("Failed to create proxy server", "error", err)
		os.Exit(1)
	}

	// Mark health server as ready now that proxy is created and plugins loaded
	healthServer.MarkReady()

	server := &http.Server{
		Addr:    addr,
		Handler: proxyServer,
	}

	go func() {
		slog.Info("Starting server", "addr", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
			// Mark health as not ready during reload
			healthServer.MarkNotReady()
			
			newCfg, err := config.Load(configFile)
			if err != nil {
				slog.Error("Failed to reload configuration", "error", err)
				healthServer.MarkReady() // Restore ready state on error
				continue
			}
			if err := proxyServer.UpdateConfig(newCfg); err != nil {
				slog.Error("Failed to update proxy configuration", "error", err)
				healthServer.MarkReady() // Restore ready state on error
				continue
			}
			
			// Mark ready again after successful reload
			healthServer.MarkReady()
			slog.Info("Configuration reloaded successfully")
		case syscall.SIGINT, syscall.SIGTERM:
			slog.Info("Shutting down server")
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			
			// Shutdown both servers
			if err := server.Shutdown(ctx); err != nil {
				slog.Error("Proxy server shutdown failed", "error", err)
			}
			if err := healthServer.Stop(); err != nil {
				slog.Error("Health server shutdown failed", "error", err)
			}
			
			cancel()
			return
		}
	}
}
