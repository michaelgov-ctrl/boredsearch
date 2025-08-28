package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
)

func (app *application) teapot(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusTeapot)
	w.Write([]byte("i'm a teapot..."))
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	ts, err := template.ParseFiles("./ui/html/pages/home.tmpl.html")
	if err != nil {
		log.Print(err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := ts.Execute(w, nil); err != nil {
		log.Print(err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

// ConnState holds the state for a single WebSocket connection.
type ConnState struct {
	Input string `json:"input"`
	Skip  int    `json:"skip"`
}

func (cs *ConnState) UnmarshalJSON(b []byte) error {
	var data struct {
		Input string `json:"input"`
		Skip  string `json:"skip"`
	}

	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}

	skip, err := strconv.Atoi(data.Skip)
	if err != nil {
		skip = 0
	}

	cs.Input = data.Input
	cs.Skip = skip

	return nil
}

func (app *application) search(w http.ResponseWriter, r *http.Request) {
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	app.addConn(conn)
	defer app.removeConn(conn)

	const windowSize = 50

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Print("read:", err)
			return
		}

		var data ConnState
		if err := json.Unmarshal(message, &data); err != nil {
			app.logger.Error(err.Error())
			continue
		}

		app.mu.Lock()
		state := app.connStates[conn]
		app.mu.Unlock()

		isNewSearch := data.Input != ""

		if isNewSearch {
			state.Input = data.Input
			state.Skip = 0
		} else {
			state.Skip = data.Skip
		}

		if state.Input == "" {
			continue
		}

		var resultsBuffer bytes.Buffer
		var moreBuffer bytes.Buffer

		emitted, more, err := app.wordTrie.WalkLeavesWindow(state.Input, state.Skip, windowSize, func(key string, value any) error {
			resultsBuffer.WriteString(fmt.Sprintf("<div class='item'>%s</div>", key))
			return nil
		})

		if err != nil {
			log.Print("trie walk:", err)
			continue
		}

		state.Skip += emitted

		if more {
			moreBuffer.WriteString(fmt.Sprintf(`
                <form id="more" ws-send hx-trigger="revealed once" hx-swap="outerHTML" hx-target="#more" hx-vals='{"skip": "%d"}'>
                    <div class='indicator'>Loading more...</div>
                </form>
            `, state.Skip))
		} else {
			moreBuffer.WriteString("<div id=\"more\"></div>")
		}

		var responseHTML string
		if isNewSearch {
			responseHTML = fmt.Sprintf(`
                <div id="results" hx-swap-oob="innerHTML">%s</div>
                <form id="more" hx-swap-oob="outerHTML">%s</form>
            `, resultsBuffer.String(), moreBuffer.String())
		} else {
			responseHTML = fmt.Sprintf(`
                <div id="results" hx-swap-oob="beforeend">%s</div>
                <form id="more" hx-swap-oob="outerHTML">%s</form>
            `, resultsBuffer.String(), moreBuffer.String())
		}

		if err = conn.WriteMessage(websocket.TextMessage, []byte(responseHTML)); err != nil {
			log.Print("write:", err)
			return
		}
	}
}

/*
func (app *application) search(w http.ResponseWriter, r *http.Request) {
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer conn.Close()

	type req struct {
		Input   string            `json:"input"`
		Skip    int               `json:"skip"`
		Headers map[string]string `json:"HEADERS"`
	}

	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			// TODO: handle err
			return
		}

		var r req
		if err := json.Unmarshal(msg, &r); err != nil || r.Input == "" {
			continue
		}

		if r.Input == "" {
			continue
		}

		walker := func(key string, value any) error {
			fmt.Println(key)
			if err = conn.WriteMessage(mt, []byte(key)); err != nil {
				// TODO: handle err
				return err
			}
			return nil
		}

		app.wordTrie.WalkLeaves(r.Input, walker)
	}
}
*/
