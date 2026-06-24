// Copyright 2026 fishfinal
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Webhook struct {
	webhookUrl string
	proxy      string
	timeout    time.Duration
	httpClient *http.Client
	// Connection pool configuration
	maxIdleConns        int
	maxIdleConnsPerHost int
	idleConnTimeout     time.Duration
}

type Option func(webhook *Webhook)

// WithTimeout sets the HTTP client timeout
func WithTimeout(duration time.Duration) Option {
	return func(webhook *Webhook) {
		webhook.timeout = duration
	}
}

// WithProxy sets the HTTP proxy for the client
func WithProxy(proxy string) Option {
	return func(webhook *Webhook) {
		webhook.proxy = proxy
	}
}

// WithConnectionPool configures connection pool parameters
func WithConnectionPool(maxIdleConns, maxIdleConnsPerHost int, idleConnTimeout time.Duration) Option {
	return func(webhook *Webhook) {
		webhook.maxIdleConns = maxIdleConns
		webhook.maxIdleConnsPerHost = maxIdleConnsPerHost
		webhook.idleConnTimeout = idleConnTimeout
	}
}

// WithMaxIdleConns sets the maximum number of idle connections
func WithMaxIdleConns(maxIdleConns int) Option {
	return func(webhook *Webhook) {
		webhook.maxIdleConns = maxIdleConns
	}
}

// WithMaxIdleConnsPerHost sets the maximum number of idle connections per host
func WithMaxIdleConnsPerHost(maxIdleConnsPerHost int) Option {
	return func(webhook *Webhook) {
		webhook.maxIdleConnsPerHost = maxIdleConnsPerHost
	}
}

// WithIdleConnTimeout sets the idle connection timeout
func WithIdleConnTimeout(idleConnTimeout time.Duration) Option {
	return func(webhook *Webhook) {
		webhook.idleConnTimeout = idleConnTimeout
	}
}

// NewWebhook creates a new Slack webhook client with the provided options
func NewWebhook(webhookUrl string, options ...Option) *Webhook {
	// Set default connection pool parameters
	slackWebhook := &Webhook{
		webhookUrl:          webhookUrl,
		timeout:             10 * time.Second,
		maxIdleConns:        100,              // Default 100 idle connections
		maxIdleConnsPerHost: 10,               // Default 10 idle connections per host
		idleConnTimeout:     90 * time.Second, // Default 90 seconds idle timeout
	}

	for _, o := range options {
		o(slackWebhook)
	}

	// Create Transport with connection pool configuration
	transport := &http.Transport{
		MaxIdleConns:        slackWebhook.maxIdleConns,
		MaxIdleConnsPerHost: slackWebhook.maxIdleConnsPerHost,
		IdleConnTimeout:     slackWebhook.idleConnTimeout,
		// Additional common configurations
		MaxConnsPerHost:    0,     // 0 means unlimited
		DisableKeepAlives:  false, // Enable Keep-Alive
		DisableCompression: false, // Enable compression
		// Timeout configurations
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// Configure proxy if provided
	if slackWebhook.proxy != "" {
		proxyURL, err := url.Parse(slackWebhook.proxy)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	// Create HTTP client
	slackWebhook.httpClient = &http.Client{
		Timeout:   slackWebhook.timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return fmt.Errorf("incorrect token (redirection)")
		},
	}

	return slackWebhook
}

// Send sends a Slack webhook message using the native net/http client
func (w *Webhook) Send(payload Payload) error {
	return w.SendWithContext(context.Background(), payload)
}

// SendWithContext sends a Slack webhook message with context support for cancellation
func (w *Webhook) SendWithContext(ctx context.Context, payload Payload) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", w.webhookUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "keep-alive")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body to ensure connection can be reused
	// Slack webhook responses are typically small, but reading is necessary
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("error sending msg. Status: %v", resp.Status)
	}

	return nil
}

// SendReceiveResponse sends a Slack webhook message and returns the raw HTTP response
// Use cases: inspecting response headers, reading response body, or checking full HTTP status
// Note: Caller must close Response.Body to reuse the connection
// Example:
//
//	resp, err := webhook.SendReceiveResponse(ctx, payload)
//	if err != nil {
//	    return err
//	}
//	defer resp.Body.Close()
//	// Inspect or read response...
func (w *Webhook) SendReceiveResponse(ctx context.Context, payload Payload) (*http.Response, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", w.webhookUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "keep-alive")
	return w.httpClient.Do(req)
}

// SendBatch sends multiple Slack webhook messages in batch
// Connection pool reuse is more effective with batch sending
func (w *Webhook) SendBatch(payloads []Payload) []error {
	var errors []error

	for _, payload := range payloads {
		if err := w.Send(payload); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}
