package main

import (
	"net/http"

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
