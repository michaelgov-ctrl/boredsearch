package main

import (
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn    *websocket.Conn
	manager *Manager
	egress  chan []byte
	buffer  chan<- string
	reset   chan struct{}
}

func NewClient(conn *websocket.Conn, m *Manager) *Client {
	c := &Client{
		conn:    conn,
		manager: m,
		egress:  make(chan []byte),
		buffer:  make(chan<- string),
		reset:   make(chan struct{}),
	}

	return c
}

var errWalkReset = errors.New("walk reset")

func (c *Client) trieWalker(key string, value any) error {
	_, ok := <-c.reset
	if ok {
		// is this race concerning?
		close(c.buffer)
		return errWalkReset
	}

	c.buffer <- key
	return nil
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

		var req Event
		if err := json.Unmarshal(payload, &req); err != nil {
			log.Println(err)
			break
		}

		if err := c.manager.routeEvent(req, c); err != nil {
			log.Println(err)
			c.egress <- []byte(err.Error())
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
