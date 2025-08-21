package main

import (
	"net/http"

	"github.com/justinas/alice"
)

// words pulled from here:
// https://github.com/dwyl/english-words/tree/master
func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	dynamic := alice.New(noSurf)

	mux.Handle("GET /{$}", dynamic.ThenFunc(app.home))
	mux.Handle("POST /{$}", dynamic.ThenFunc(app.search))

	// meme endpoint
	mux.HandleFunc("/teapot", app.teapot)

	standard := alice.New(app.recoverPanic, app.logRequest, commonHeaders)

	return standard.Then(mux)
}
