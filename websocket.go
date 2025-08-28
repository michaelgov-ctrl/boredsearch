package main

import (
	"net/http"
	"sync"

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

	// A map to store the state for each connection.
	// Key: *websocket.Conn, Value: *ConnState
	connStates = make(map[*websocket.Conn]*ConnState)
	mu         sync.Mutex // Mutex to protect concurrent access to connStates
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
