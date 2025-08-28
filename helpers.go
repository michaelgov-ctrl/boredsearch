package main

import (
	"bytes"
	"fmt"
	"net/http"
)

func (app *application) serverError(w http.ResponseWriter, r *http.Request, err error) {
	var (
		method = r.Method
		uri    = r.URL.RequestURI()
	)

	app.logger.Error(err.Error(), "method", method, "uri", uri)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func newRespHTML(isNewSearch bool, resBuf, moreBuf *bytes.Buffer) []byte {
	if isNewSearch {
		return fmt.Appendf([]byte{}, `
                <div id="results" hx-swap-oob="innerHTML">%s</div>
                <form id="more" hx-swap-oob="outerHTML">%s</form>
            `, resBuf.Bytes(), moreBuf.Bytes())
	}

	return fmt.Appendf([]byte{}, `
			<div id="results" hx-swap-oob="beforeend">%s</div>
			<form id="more" hx-swap-oob="outerHTML">%s</form>
		`, resBuf.Bytes(), moreBuf.Bytes())
}

func bufferMoreHTML(more bool, skip int) string {
	if more {
		return fmt.Sprintf(`
			<form id="more" ws-send hx-trigger="revealed once" hx-swap="outerHTML" hx-target="#more" hx-vals='{"skip": "%d"}'>
				<div class='indicator'>Loading more...</div>
			</form>
		`, skip)
	}

	return "<div id=\"more\"></div>"
}

func bufferResultsHTML(key string) string {
	return fmt.Sprintf("<div class='item'>%s</div>", key)
}
