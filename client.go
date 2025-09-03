package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const windowSize = 50

type mailbox struct {
	ctx    context.Context
	cancel context.CancelFunc
	ch     chan string
}

type Client struct {
	conn    *websocket.Conn
	manager *Manager
	egress  chan []byte
	mu      sync.Mutex
	active  *mailbox
}

func NewClient(conn *websocket.Conn, m *Manager) *Client {
	c := &Client{
		conn:    conn,
		manager: m,
		egress:  make(chan []byte, 8),
	}

	c.startSearch("")

	return c
}

func (c *Client) setMailbox() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.active != nil {
		c.active.cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.active = &mailbox{
		ctx:    ctx,
		cancel: cancel,
		ch:     make(chan string, windowSize),
	}
}

func (c *Client) startSearch(input string) {
	c.setMailbox()
	go c.manager.searchWithCtx(c.active.ctx, input, c.active.ch)
	c.sendHTML(Search)
}

func (c *Client) readEvents() {
	defer func() {
		c.manager.removeClient(c)
	}()

	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		c.manager.logger.Error(err.Error())
		return
	}

	c.conn.SetReadLimit(512)
	c.conn.SetPongHandler(c.pongHandler)

	for {
		_, payload, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.manager.logger.Error(err.Error())
			}
			break
		}

		var req Event
		if err := json.Unmarshal(payload, &req); err != nil {
			c.manager.logger.Error(err.Error())
			break
		}

		if err := c.manager.routeEvent(req, c); err != nil {
			c.manager.logger.Error(err.Error())
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
					c.manager.logger.Error(err.Error())
				}
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				c.manager.logger.Error(err.Error())
				return
			}
		case <-ticker.C:
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte(``)); err != nil {
				c.manager.logger.Error(err.Error())
				return
			}
		}
	}
}

func (c *Client) pongHandler(pongMsg string) error {
	return c.conn.SetReadDeadline(time.Now().Add(pongWait))
}

func (c *Client) collectLeaves() []string {
	c.mu.Lock()
	mb := c.active
	c.mu.Unlock()
	if mb == nil {
		return nil
	}

	var words []string
	for len(words) < windowSize {
		s, ok := <-mb.ch
		if !ok {
			break
		}

		words = append(words, fmt.Sprintf("<div class='item'>%s</div>", s))
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

func (c *Client) sendHTML(ho HtmxOob) {
	words := c.collectLeaves()
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
		html += `<div id="more"></div>`
	}

	c.egress <- []byte(html)
}
