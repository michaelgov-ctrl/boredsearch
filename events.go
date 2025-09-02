package main

import (
	"encoding/json"
	"strconv"
)

type Event struct {
	Headers HtmxHeaders     `json:"HEADERS"`
	Payload json.RawMessage `json:"payload"`
}

type EventHandler func(event Event, c *Client) error

const (
	EventSearch = "search"
	EventMore   = "more"
)

type SearchEvent struct {
	Input string `json:"input"`
}

type MoreEvent struct{} // for now I need no further data

type HtmxHeaders struct {
	CurrentURL  string `json:"HX-Current-URL"`
	Request     bool   `json:"HX-Request"`
	Target      string `json:"HX-Target"`
	Trigger     string `json:"HX-Trigger"`
	TriggerName string `json:"HX-Trigger-Name"`
}

func (hh *HtmxHeaders) UnmarshalJSON(b []byte) error {
	var raw map[string]string
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	hh.CurrentURL = raw["HX-Current-URL"]
	hh.Target = raw["HX-Target"]
	hh.Trigger = raw["HX-Trigger"]
	hh.TriggerName = raw["HX-Trigger-Name"]

	req, err := strconv.ParseBool(raw["HX-Request"])
	if err != nil {
		return err
	}
	hh.Request = req

	return nil
}
