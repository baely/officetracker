package utils

import (
	"bytes"
	"encoding/json"
	"net/http"
	url2 "net/url"
)

func NewRequest(host, method, path string, body map[string]interface{}) (*http.Request, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url, err := url2.JoinPath(host, path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}

	return req, nil
}
