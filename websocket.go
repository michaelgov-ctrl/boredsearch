package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
)

var (
	websocketUpgrader = websocket.Upgrader{
		CheckOrigin:     checkOrigin,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	// TODO: update this for proxy
	AllowedOrigins = []string{
		"ws://localhost:4040",
		"http://localhost:4040",
	}
)

// TODO: update this for proxy
func checkOrigin(r *http.Request) bool {
	return true
	/*
		origin := r.Header.Get("Origin")
		for _, o := range AllowedOrigins {
			if origin == o {
				return true
			}
		}

		return false
	*/
}

type HtmxHeaders struct {
	CurrentURL  string `json:"HX-Current-URL"`
	Request     bool   `json:"HX-Request"`
	Target      string `json:"HX-Target"`
	Trigger     string `json:"HX-Trigger"`
	TriggerName string `json:"HX-Trigger-Name"`
}

func (hh *HtmxHeaders) UnmarshalJSON(b []byte) error {
	var raw map[string]string
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	hh.CurrentURL = raw["HX-Current-URL"]
	hh.Target = raw["HX-Target"]
	hh.Trigger = raw["HX-Trigger"]
	hh.TriggerName = raw["HX-Trigger-Name"]

	req, err := strconv.ParseBool(raw["HX-Request"])
	if err != nil {
		return err
	}
	hh.Request = req

	return nil
}

type ConnState struct {
	Input   string      `json:"input"`
	Skip    int         `json:"skip"`
	Headers HtmxHeaders `json:"HEADERS"`
}

func (cs *ConnState) UnmarshalJSON(b []byte) error {
	var data struct {
		Input   string      `json:"input"`
		Skip    string      `json:"skip"`
		Headers HtmxHeaders `json:"HEADERS"`
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
	cs.Headers = data.Headers

	return nil
}

func (app *application) addConn(conn *websocket.Conn) {
	app.mu.Lock()
	defer app.mu.Unlock()

	app.connStates[conn] = &ConnState{
		Input: "",
		Skip:  0,
	}
}

func (app *application) removeConn(conn *websocket.Conn) {
	conn.Close()

	app.mu.Lock()
	defer app.mu.Unlock()
	delete(app.connStates, conn)
}

func (app *application) getConnState(conn *websocket.Conn) *ConnState {
	app.mu.Lock()
	defer app.mu.Unlock()

	return app.connStates[conn]
}
