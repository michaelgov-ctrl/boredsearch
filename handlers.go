package main

import (
	"html/template"
	"log"
	"net/http"
)

const windowSize = 50

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

func (app *application) search(w http.ResponseWriter, r *http.Request) {
	_, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	/*
		var resultsBuffer bytes.Buffer
		var moreBuffer bytes.Buffer

		emitted, more, err := app.wordTrie.WalkLeavesWindow(state.Input, state.Skip, windowSize, func(key string, value any) error {
			resultsBuffer.WriteString(bufferResultsHTML(key))
			return nil
		})
		if err != nil {
			app.logger.Error(err.Error())
			continue
		}

		state.Skip += emitted
		moreBuffer.WriteString(bufferMoreHTML(more, state.Skip))

		respHTML := newRespHTML(isNewSearch, &resultsBuffer, &moreBuffer)
		if err = conn.WriteMessage(websocket.TextMessage, respHTML); err != nil {
			app.logger.Error(err.Error())
			return
		}
	*/
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
