package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

var (
	ntfyEndpoint    = os.Getenv("NTFY_ENDPOINT")
	backendEndpoint = os.Getenv("BACKEND_ENDPOINT")
)

type Action struct {
	Action string `json:"action"`
	Label  string `json:"label"`
	Url    string `json:"url"`
	Clear  bool   `json:"clear,omitempty"`
}

func (a Action) asHeader() string {
	s := a.Action + ", " + a.Label + ", " + a.Url

	if a.Clear {
		s += ", " + "clear=true"
	} else {
		s += "," + "clear=false"
	}

	return s
}

type Payload struct {
	Topic   string   `json:"topic"`
	Message string   `json:"message"`
	Actions []Action `json:"actions"`
}

func NewPayload(topic, message string) *Payload {
	return &Payload{
		Topic:   topic,
		Message: message,
	}
}

func (p *Payload) AddAction(label, url string) {
	a := Action{
		Action: "view",
		Label:  label,
		Url:    url,
		Clear:  false,
	}

	p.Actions = append(p.Actions, a)
}

func (p *Payload) Send() error {
	b := []byte(p.Message)

	err := send(map[string][]string{"Actions": {p.formActions()}}, b)
	if err != nil {
		return err
	}

	return nil
}

func (p *Payload) formActions() string {
	var actions []string
	for _, action := range p.Actions {
		actions = append(actions, action.asHeader())
	}
	h := strings.Join(actions, ";")
	return h
}

func send(headers map[string][]string, msg []byte) error {
	r := bytes.NewReader(msg)
	req, err := http.NewRequest(http.MethodPost, ntfyEndpoint, r)
	if err != nil {
		return err
	}

	for header, values := range headers {
		for _, value := range values {
			req.Header.Add(header, value)
		}
	}

	slog.Debug(fmt.Sprintf("sending request: %v", req))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send notification: %v", resp.Status)
	}

	return nil
}
