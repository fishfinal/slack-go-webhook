// webhook_test.go
package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// WebhookTestSuite defines the test suite for Webhook
type WebhookTestSuite struct {
	suite.Suite
	server  *httptest.Server
	webhook *Webhook
	payload Payload
}

// SetupSuite runs once before all tests
func (s *WebhookTestSuite) SetupSuite() {
	// Create a test server that handles Slack webhook requests
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		s.Equal("POST", r.Method)

		// Verify content type
		s.Equal("application/json", r.Header.Get("Content-Type"))

		// Verify connection header
		s.Equal("keep-alive", r.Header.Get("Connection"))

		// Decode and verify payload
		var payload Payload
		err := json.NewDecoder(r.Body).Decode(&payload)
		s.NoError(err)

		// Simulate Slack responses
		if r.URL.Path == "/success" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		} else if r.URL.Path == "/error" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid_payload"))
		} else if r.URL.Path == "/rate_limit" {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("rate_limited"))
		} else if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/success", http.StatusFound)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		}
	}))
}

// TearDownSuite runs once after all tests
func (s *WebhookTestSuite) TearDownSuite() {
	s.server.Close()
}

// SetupTest runs before each test
func (s *WebhookTestSuite) SetupTest() {
	// Create webhook with test server URL
	s.webhook = NewWebhook(
		s.server.URL+"/success",
		WithTimeout(5*time.Second),
		WithConnectionPool(10, 5, 30*time.Second),
	)

	// Prepare test payload
	s.payload = Payload{
		Text:     "Test message",
		Username: "TestBot",
		Channel:  "#test",
		Attachments: []Attachment{
			{
				Title: stringPtr("Test Title"),
				Text:  stringPtr("Test Text"),
				Color: stringPtr("#36a64f"),
				Fields: []*Field{
					{Title: "Field 1", Value: "Value 1", Short: true},
					{Title: "Field 2", Value: "Value 2", Short: false},
				},
			},
		},
	}
}

// TestWebhookSuite runs the test suite
func TestWebhookSuite(t *testing.T) {
	suite.Run(t, new(WebhookTestSuite))
}

// TestSend tests the Send method
func (s *WebhookTestSuite) TestSend() {
	err := s.webhook.Send(s.payload)
	s.NoError(err)
}

// TestSendWithContext tests the SendWithContext method
func (s *WebhookTestSuite) TestSendWithContext() {
	ctx := context.Background()
	err := s.webhook.SendWithContext(ctx, s.payload)
	s.NoError(err)
}

// TestSendWithContextCancellation tests context cancellation
func (s *WebhookTestSuite) TestSendWithContextCancellation() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := s.webhook.SendWithContext(ctx, s.payload)
	s.Error(err)
	s.Contains(err.Error(), "context canceled")
}

// TestSendWithContextTimeout tests context timeout
func (s *WebhookTestSuite) TestSendWithContextTimeout() {
	// Create a slow server
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	webhook := NewWebhook(
		slowServer.URL,
		WithTimeout(1*time.Second),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := webhook.SendWithContext(ctx, s.payload)
	s.Error(err)
}

// TestSendReceiveResponse tests the SendReceiveResponse method
func (s *WebhookTestSuite) TestSendReceiveResponse() {
	ctx := context.Background()
	resp, err := s.webhook.SendReceiveResponse(ctx, s.payload)
	s.NoError(err)
	s.NotNil(resp)
	s.Equal(http.StatusOK, resp.StatusCode)

	// Ensure we close the response body
	defer resp.Body.Close()
}

// TestSendReceiveResponseWithError tests error response
func (s *WebhookTestSuite) TestSendReceiveResponseWithError() {
	webhook := NewWebhook(s.server.URL + "/error")

	ctx := context.Background()
	resp, err := webhook.SendReceiveResponse(ctx, s.payload)
	s.NoError(err) // The request itself succeeds
	s.NotNil(resp)
	s.Equal(http.StatusBadRequest, resp.StatusCode)
	defer resp.Body.Close()
}

// TestSendReceiveResponseWithRateLimit tests rate limit response
func (s *WebhookTestSuite) TestSendReceiveResponseWithRateLimit() {
	webhook := NewWebhook(s.server.URL + "/rate_limit")

	ctx := context.Background()
	resp, err := webhook.SendReceiveResponse(ctx, s.payload)
	s.NoError(err)
	s.NotNil(resp)
	s.Equal(http.StatusTooManyRequests, resp.StatusCode)
	defer resp.Body.Close()
}

// TestSendWithError tests Send with error response
func (s *WebhookTestSuite) TestSendWithError() {
	webhook := NewWebhook(s.server.URL + "/error")

	err := webhook.Send(s.payload)
	s.Error(err)
	s.Contains(err.Error(), "Status: 400 Bad Request")
}

// TestSendBatch tests the SendBatch method
func (s *WebhookTestSuite) TestSendBatch() {
	payloads := []Payload{
		s.payload,
		{Text: "Message 2", Username: "Bot2"},
		{Text: "Message 3", Username: "Bot3"},
	}

	errors := s.webhook.SendBatch(payloads)
	s.Empty(errors) // No errors expected
}

// TestSendBatchWithErrors tests SendBatch with some errors
func (s *WebhookTestSuite) TestSendBatchWithErrors() {
	webhook := NewWebhook(s.server.URL + "/error")

	payloads := []Payload{
		s.payload,
		{Text: "Message 2"},
		{Text: "Message 3"},
	}

	errors := webhook.SendBatch(payloads)
	s.NotEmpty(errors)
	s.Len(errors, 3) // All should fail
}

// TestNewWebhookWithOptions tests various Option configurations
func (s *WebhookTestSuite) TestNewWebhookWithOptions() {
	tests := []struct {
		name    string
		options []Option
		want    func(*Webhook) bool
	}{
		{
			name:    "default options",
			options: []Option{},
			want: func(w *Webhook) bool {
				return w.timeout == 10*time.Second &&
					w.maxIdleConns == 100 &&
					w.maxIdleConnsPerHost == 10 &&
					w.idleConnTimeout == 90*time.Second
			},
		},
		{
			name: "with timeout",
			options: []Option{
				WithTimeout(30 * time.Second),
			},
			want: func(w *Webhook) bool {
				return w.timeout == 30*time.Second
			},
		},
		{
			name: "with proxy",
			options: []Option{
				WithProxy("http://proxy.example.com:8080"),
			},
			want: func(w *Webhook) bool {
				return w.proxy == "http://proxy.example.com:8080"
			},
		},
		{
			name: "with connection pool",
			options: []Option{
				WithConnectionPool(200, 20, 120*time.Second),
			},
			want: func(w *Webhook) bool {
				return w.maxIdleConns == 200 &&
					w.maxIdleConnsPerHost == 20 &&
					w.idleConnTimeout == 120*time.Second
			},
		},
		{
			name: "with max idle conns",
			options: []Option{
				WithMaxIdleConns(50),
			},
			want: func(w *Webhook) bool {
				return w.maxIdleConns == 50
			},
		},
		{
			name: "with max idle conns per host",
			options: []Option{
				WithMaxIdleConnsPerHost(15),
			},
			want: func(w *Webhook) bool {
				return w.maxIdleConnsPerHost == 15
			},
		},
		{
			name: "with idle conn timeout",
			options: []Option{
				WithIdleConnTimeout(60 * time.Second),
			},
			want: func(w *Webhook) bool {
				return w.idleConnTimeout == 60*time.Second
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			webhook := NewWebhook(s.server.URL+"/success", tt.options...)
			s.True(tt.want(webhook), "Option configuration failed")
		})
	}
}

// TestRedirectPolicy tests that redirects are not followed
func (s *WebhookTestSuite) TestRedirectPolicy() {
	webhook := NewWebhook(s.server.URL + "/redirect")

	err := webhook.Send(s.payload)
	s.Error(err)
	s.Contains(err.Error(), "incorrect token")
}

// TestPayloadSerialization tests payload JSON marshaling
func (s *WebhookTestSuite) TestPayloadSerialization() {
	payload := Payload{
		Text:     "Test",
		Username: "Bot",
		Channel:  "#general",
		Attachments: []Attachment{
			{
				Title: stringPtr("Title"),
				Text:  stringPtr("Text"),
				Color: stringPtr("#ff0000"),
				Fields: []*Field{
					{Title: "F1", Value: "V1", Short: true},
				},
			},
		},
	}

	data, err := json.Marshal(payload)
	s.NoError(err)

	var decoded Payload
	err = json.Unmarshal(data, &decoded)
	s.NoError(err)

	s.Equal(payload.Text, decoded.Text)
	s.Equal(payload.Username, decoded.Username)
	s.Equal(payload.Channel, decoded.Channel)
	s.Len(decoded.Attachments, 1)
	s.Equal("Title", *decoded.Attachments[0].Title)
	s.Len(decoded.Attachments[0].Fields, 1)
}

// TestAttachmentHelpers tests the AddField and AddAction helper methods
func (s *WebhookTestSuite) TestAttachmentHelpers() {
	attachment := &Attachment{
		Title: stringPtr("Test"),
	}

	// Test AddField
	field := Field{Title: "Field1", Value: "Value1", Short: true}
	attachment.AddField(field)
	s.Len(attachment.Fields, 1)
	s.Equal("Field1", attachment.Fields[0].Title)

	// Test AddAction
	action := Action{
		Type:  "button",
		Text:  "Click me",
		Url:   "https://example.com",
		Style: "primary",
	}
	attachment.AddAction(action)
	s.Len(attachment.Actions, 1)
	s.Equal("button", attachment.Actions[0].Type)
}

// TestWebhookWithInvalidProxy tests proxy parsing error handling
func (s *WebhookTestSuite) TestWebhookWithInvalidProxy() {
	webhook := NewWebhook(
		s.server.URL+"/success",
		WithProxy(":invalid:proxy:"),
	)

	// Proxy parsing should fail silently and not set proxy
	s.Equal(":invalid:proxy:", webhook.proxy)

	// Should still work
	err := webhook.Send(s.payload)
	s.NoError(err)
}

// TestSendWithNilPayload tests sending with minimal payload
func (s *WebhookTestSuite) TestSendWithMinimalPayload() {
	payload := Payload{
		Text: "Minimal message",
	}

	err := s.webhook.Send(payload)
	s.NoError(err)
}

// Helper function for string pointers
func stringPtr(s string) *string {
	return &s
}

// Benchmark tests for performance
func BenchmarkSend(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	webhook := NewWebhook(server.URL)
	payload := Payload{Text: "Benchmark message"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = webhook.Send(payload)
	}
}

func BenchmarkSendBatch(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	webhook := NewWebhook(server.URL)
	payloads := make([]Payload, 10)
	for i := range payloads {
		payloads[i] = Payload{Text: fmt.Sprintf("Message %d", i)}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = webhook.SendBatch(payloads)
	}
}
