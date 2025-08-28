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
