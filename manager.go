package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
)

type Manager struct {
	clients  map[*Client]bool
	mu       sync.RWMutex
	handlers map[string]EventHandler
	wordTrie *Trie
}

func NewManager() (*Manager, error) {
	file, err := os.Open("words.txt")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	trie, err := NewTrie().LoadFromFile(file)
	if err != nil {
		return nil, err
	}

	m := &Manager{
		clients:  make(map[*Client]bool),
		mu:       sync.RWMutex{},
		handlers: make(map[string]EventHandler),
		wordTrie: trie,
	}

	m.registerEventHandlers()

	return m, nil
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

	m.clients[c] = true
}

func (m *Manager) removeClient(c *Client) {
	c.conn.Close()

	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, c)
}

func (m *Manager) registerEventHandlers() {
	m.handlers[EventSearch] = m.searchHandler
	m.handlers[EventMore] = m.moreHandler
}

func (m *Manager) routeEvent(event Event, c *Client) error {
	log.Println(event.Headers.Trigger)

	handler, ok := m.handlers[event.Headers.Trigger]
	if !ok {
		return errors.New("there is no such event type")
	}

	if err := handler(event, c); err != nil {
		return err
	}

	return nil
}

func (m *Manager) searchHandler(event Event, c *Client) error {
	var searchEvent SearchEvent
	if err := json.Unmarshal(event.Payload, &searchEvent); err != nil {
		return fmt.Errorf("bad payload in request: %v", err)
	}

	go func() {
		c.reset <- struct{}{}
		m.search(searchEvent.Input, c)
	}()

	c.sendSearchHTML(searchEvent.Input)

	return nil
}

func (m *Manager) search(str string, c *Client) {
	defer func() {
		c.buffer = make(chan string)
		c.once.Reset()
	}()

	if err := c.manager.wordTrie.WalkLeaves(str, c.trieWalker); err != nil {
		if errors.Is(err, errWalkReset) {
			return
		}
		log.Printf("error for: %v, error: %v", &c, err)
	}
}

func (m *Manager) moreHandler(event Event, c *Client) error {
	c.sendMoreHTML()
	return nil
}
