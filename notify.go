package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
)

const (
	ntfyEndpoint = "https://ntfy.sh/"
)

var (
	ntfyTopic       = os.Getenv("NTFY_TOPIC")
	backendEndpoint = os.Getenv("BACKEND_ENDPOINT")
)

type Payload struct {
	Topic   string `json:"topic"`
	Message string `json:"message"`
	Title   string `json:"title"`
	Click   string `json:"click"`
}

func NewPayload(title, message, click string) *Payload {
	return &Payload{
		Topic:   ntfyTopic,
		Title:   title,
		Message: message,
		Click:   click,
	}
}

func (p *Payload) Send() error {
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	err = send(b)
	if err != nil {
		return err
	}

	return nil
}

func send(msg []byte) error {
	r := bytes.NewReader(msg)
	_, err := http.Post(ntfyEndpoint, "application/json", r)
	if err != nil {
		return err
	}

	return nil
}
