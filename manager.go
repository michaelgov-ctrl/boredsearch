package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
)

type Manager struct {
	clients  map[*Client]bool
	mu       sync.RWMutex
	handlers map[string]EventHandler
	wordTrie *Trie
	logger   *slog.Logger
}

// words pulled from here:
// https://github.com/dwyl/english-words/tree/master

func NewManager(logger *slog.Logger) (*Manager, error) {
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
		logger:   logger,
	}

	m.registerEventHandlers()

	return m, nil
}

func (m *Manager) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		m.logger.Debug(fmt.Sprintf("upgrade: %v", err))
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
	m.logger.Debug(fmt.Sprintf("routing event: %s", event.Headers.Trigger))

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
	c.startSearch(searchEvent.Input)
	return nil
}

func (m *Manager) searchWithCtx(ctx context.Context, prefix string, out chan<- string) {
	defer close(out)

	walker := func(key string, _ any) error {
		select {
		case <-ctx.Done():
			return context.Canceled
		case out <- key:
			return nil
		}
	}

	if err := m.wordTrie.WalkLeaves(prefix, walker); err != nil && !errors.Is(err, context.Canceled) {
		m.logger.Error(err.Error())
	}
}

func (m *Manager) moreHandler(event Event, c *Client) error {
	c.sendHTML(More)
	return nil
}
