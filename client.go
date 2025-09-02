package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const windowSize = 50

type Client struct {
	conn    *websocket.Conn
	manager *Manager
	egress  chan []byte
	buffer  chan string
	reset   chan struct{}
	once    ResettableOnce
}

func NewClient(conn *websocket.Conn, m *Manager) *Client {
	c := &Client{
		conn:    conn,
		manager: m,
		egress:  make(chan []byte),
		buffer:  make(chan string, windowSize),
		reset:   make(chan struct{}),
		once:    ResettableOnce{},
	}

	go c.manager.search("", c)
	go c.sendSearchHTML("")

	return c
}

var errWalkReset = errors.New("walk reset")

func (c *Client) trieWalker(key string, value any) error {
	fmt.Println(key)
	select {
	case <-c.reset:
		c.once.Do(func() {
			close(c.buffer)
		})
		return errWalkReset
	case c.buffer <- key:
		return nil
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

func (c *Client) collectLeaves(prefix string) []string {
	var words []string

	for range windowSize {
		s := <-c.buffer
		if !strings.HasPrefix(s, prefix) { // naively drain channel till i think of something better
			continue
		}

		words = append(words, fmt.Sprintf(
			"<div class='item'>%s</div>", s,
		))
	}

	return words
}

func (c *Client) sendSearchHTML(prefix string) {
	words := c.collectLeaves(prefix)
	wordsBlock := strings.Join(words, "\n")
	html := fmt.Sprintf("<div id=\"results\" hx-swap-oob=\"innerHTML\">%s</div>\n", wordsBlock)

	if len(words) == windowSize {
		html += `
				<form id="more" ws-send hx-trigger="revealed once" hx-swap="outerHTML" hx-target="#more"'>
					<div class='indicator'>Loading more...</div>
				</form>
				`
	} else {
		html += "<div id=\"more\"></div>"
	}

	c.egress <- []byte(html)
}

func (c *Client) sendMoreHTML() {
	words := c.collectLeaves("")
	wordsBlock := strings.Join(words, "\n")
	html := fmt.Sprintf("<div id=\"results\" hx-swap-oob=\"beforeend\">%s</div>\n", wordsBlock)

	if len(words) == windowSize {
		html += `
				<form id="more" ws-send hx-trigger="revealed once" hx-swap="outerHTML" hx-target="#more"'>
					<div class='indicator'>Loading more...</div>
				</form>
				`
	} else {
		html += "<div id=\"more\"></div>"
	}

	c.egress <- []byte(html)
}

type ResettableOnce struct {
	mu   sync.Mutex
	done bool
}

func (o *ResettableOnce) Do(f func()) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.done {
		f()
		o.done = true
	}
}

func (o *ResettableOnce) Reset() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.done = false
}
