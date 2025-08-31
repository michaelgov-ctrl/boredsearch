package main

import (
	"log"
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

// words pulled from here:
// https://github.com/dwyl/english-words/tree/master
func main() {
	man, err := NewManager()
	if err != nil {
		log.Fatal(err)
	}

	app := &application{
		config:  config{addr: ":4040"},
		logger:  slog.New(slog.NewTextHandler(os.Stdout, nil)),
		manager: man,
	}

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
