package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"
)

type config struct {
	addr string
}

type application struct {
	config  config
	logger  *slog.Logger
	manager *Manager
}

func main() {
	app := &application{
		config: config{addr: ":4040"},
		logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}

	man, err := NewManager(app.logger)
	if err != nil {
		app.logger.Error(err.Error())
		os.Exit(1)
	}

	app.manager = man

	srv := &http.Server{
		Addr:         app.config.addr,
		Handler:      app.routes(),
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	app.logger.Info("starting server", "addr", app.config.addr)

	if err := srv.ListenAndServe(); err != nil {
		app.logger.Error(err.Error())
		os.Exit(1)
	}
}
