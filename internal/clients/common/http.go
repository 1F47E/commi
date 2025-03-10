package common

import (
	"bytes"
	"net/http"
	"time"
)

const (
	DefaultTimeout    = 10 * time.Second
	DefaultMaxRetries = 3
	DefaultRetryDelay = 1 * time.Second
)

type ClientConfig struct {
	Timeout    time.Duration
	MaxRetries int
	RetryDelay time.Duration
	Headers    map[string]string
}

func DefaultConfig() ClientConfig {
	return ClientConfig{
		Timeout:    DefaultTimeout,
		MaxRetries: DefaultMaxRetries,
		RetryDelay: DefaultRetryDelay,
		Headers:    make(map[string]string),
	}
}

func NewHTTPClient(config ClientConfig) *http.Client {
	return &http.Client{
		Timeout: config.Timeout,
	}
}

func NewRequest(method, url string, body []byte, config ClientConfig) (*http.Request, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	// Set common headers
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	return req, nil
}
