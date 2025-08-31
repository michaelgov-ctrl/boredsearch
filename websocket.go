package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

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

type EventMessage struct {
	Input   string            `json:"input"`
	Skip    int               `json:"skip"`
	Headers map[string]string `json:"headers"`
}

func (em *EventMessage) UnmarshalJSON(b []byte) error {
	var data struct {
		Input   string            `json:"input"`
		Skip    string            `json:"skip"`
		Headers map[string]string `json:"headers"`
	}

	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}

	skip, err := strconv.Atoi(data.Skip)
	if err != nil {
		skip = 0
	}

	em.Input = data.Input
	em.Skip = skip
	em.Headers = data.Headers

	return nil
}

var (
	pongWait     = 10 * time.Second
	pingInterval = (pongWait * 9) / 10 // 90% of pongWait
)

type MessageType int

const (
	Input = iota
	Skip
	Error
)

type ClientState struct {
	Type    MessageType
	Message any
}

type Client struct {
	conn    *websocket.Conn
	manager *Manager
	egress  chan []byte
}

func NewClient(conn *websocket.Conn, m *Manager) *Client {
	return &Client{
		conn:    conn,
		manager: m,
		egress:  make(chan []byte),
	}
}

func (c *Client) readEvents() {
	defer func() {
		c.manager.removeClient(c)
	}()

	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Println(err)
		return
	}

	c.conn.SetReadLimit(512)
	c.conn.SetPongHandler(c.pongHandler)

	for {
		_, payload, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println(err)
			}
			break
		}

		var req EventMessage
		if err := json.Unmarshal(payload, &req); err != nil {
			log.Println(err)
			break
		}

		if err := c.manager.routeEvent(req, c); err != nil {
			log.Println(err)
			c.egress <- err.Error()
		}
	}
}

func (c *Client) writeEvents() {
	defer func() {
		c.manager.removeClient(c)
	}()

	ticker := time.NewTicker(pingInterval)

	for {
		// bottle necking to prevent abuse of concurrency from client
		select {
		case msg, ok := <-c.egress:
			if !ok {
				// if egress is broken notify client & close
				if err := c.conn.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Println(err)
				}
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Println(err)
				return
			}
		case <-ticker.C:
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte(``)); err != nil {
				log.Println(err)
				return
			}
		}
	}
}

func (c *Client) pongHandler(pongMsg string) error {
	return c.conn.SetReadDeadline(time.Now().Add(pongWait))
}

type Manager struct {
	clients map[*Client]ClientState
	mu      sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		clients: make(map[*Client]ClientState),
		mu:      sync.RWMutex{},
	}
}

func (m *Manager) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	client := NewClient(conn, m)

	m.addClient(client)

	go client.readEvents()
	go client.writeEvents()
}

func (m *Manager) addClient(c *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.clients[c] = ClientState{}
}

func (m *Manager) removeClient(c *Client) {
	c.conn.Close()

	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, c)
}

func (m *Manager) getConnState(c *Client) ClientState {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.clients[c]
}
