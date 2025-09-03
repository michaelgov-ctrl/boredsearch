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

var errWalkReset = errors.New("walk reset")

type Client struct {
	conn    *websocket.Conn
	manager *Manager
	egress  chan []byte
	buffer  chan string
	reset   chan struct{}
	once    ResettableOnce
	mu      sync.Mutex
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
	go c.sendHTML("", Search)

	return c
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

func (c *Client) trieWalker(key string, value any) error {
	//fmt.Println(key)
	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case <-c.reset:
		//c.once.Do(func() {
		//close(c.buffer)
		//})
		c.CloseBuffer()
		return errWalkReset
	case c.buffer <- key:
		return nil
	}
}

func (c *Client) collectLeaves(prefix string) []string {
	var words []string

	for i := 0; i < windowSize; {
		s := <-c.buffer
		if !strings.HasPrefix(s, prefix) { // naively drain channel till i think of something better
			continue
		}

		words = append(words, fmt.Sprintf(
			"<div class='item'>%s</div>", s,
		))

		i++
	}

	return words
}

type HtmxOob int

const (
	Search HtmxOob = iota
	More
)

func (ho HtmxOob) String() string {
	switch ho {
	case Search:
		return "innerHTML"
	case More:
		return "beforeend"
	default:
		return fmt.Sprintf("unknown state (%d)", ho)
	}
}

func (c *Client) sendHTML(prefix string, ho HtmxOob) {
	words := c.collectLeaves(prefix)
	wordsBlock := strings.Join(words, "\n")
	html := fmt.Sprintf("<div id=\"results\" hx-swap-oob=\"%s\">%s</div>\n", ho, wordsBlock)

	if len(words) == windowSize {
		html += `
				<form id="more"
					hx-ext="ws-wrap-payload"
					ws-send
					hx-trigger="revealed once"
					hx-swap="outerHTML"
					hx-target="#more"
					hx-include="#input">
					<div class='indicator'>Loading more...</div>
				</form>
				`
	} else {
		html += "<div id=\"more\"></div>"
	}

	c.egress <- []byte(html)
}

func (c *Client) CloseBuffer() {
	close(c.buffer)
}

func (c *Client) ResetBuffer() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.buffer = make(chan string)
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
