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
	config   config
	logger   *slog.Logger
	wordTrie *Trie
}

func main() {
	file, err := os.Open("words.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	trie, err := NewTrie().LoadFromFile(file)
	if err != nil {
		log.Fatal(err)
	}

	app := &application{
		config:   config{addr: ":4040"},
		logger:   slog.New(slog.NewTextHandler(os.Stdout, nil)),
		wordTrie: trie,
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

/*
	for _, s := range []string{
		"refoundation",
		"galravage",
		"antiproductive",
		"unctioneer",
		"Zwinglianist",
	} {
		fmt.Println(app.wordTrie.Get(s))
	}

	walker := func(key string, value any) error {
		fmt.Println(key)
		return nil
	}

	for _, s := range []string{
		"Zun",
		"kata",
		"AAASA",
	} {
		app.wordTrie.WalkLeaves(s, walker)
	}

	time.Sleep(10 * time.Minute)
*/
