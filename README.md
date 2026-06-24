# Slack Webhook

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/fishfinal/slack-webhook)](https://goreportcard.com/report/github.com/fishfinal/slack-webhook)
[![codecov](https://codecov.io/gh/fishfinal/slack-go-webhook/branch/main/graph/badge.svg)](https://codecov.io/gh/fishfinal/slack-go-webhook)

A lightweight, high-performance Slack webhook client for Go with connection pooling, context support, and comprehensive configuration options.

> **Note:** This library is a complete rewrite and extension of [ashwanthkumar/slack-go-webhook](https://github.com/ashwanthkumar/slack-go-webhook). It maintains the same core API for backward compatibility while adding significant performance improvements, new features, and better Go practices. Special thanks to [@ashwanthkumar](https://github.com/ashwanthkumar) for the original implementation.

## Features

- 🚀 **High Performance** - Connection pooling with configurable idle connections
- ⏱️ **Context Support** - Full context support for cancellation and timeouts
- 🔧 **Flexible Configuration** - Functional options pattern for easy customization
- 🌐 **Proxy Support** - HTTP/HTTPS proxy support
- 📦 **Batch Sending** - Send multiple messages efficiently
- 🧪 **Tested** - Comprehensive test suite with testify
- 📝 **Type Safe** - Strongly typed Slack message structures
- 🔒 **Keep-Alive** - HTTP keep-alive for connection reuse
- 🎯 **100% Compatible** - Fully compatible with the original library's API

## Credits & Acknowledgments

This project is built upon the work of:

- [ashwanthkumar/slack-go-webhook](https://github.com/ashwanthkumar/slack-go-webhook) - The original Slack webhook library that inspired this rewrite

### Key Improvements Over the Original

| Feature | Original | This Library |
|---------|----------|--------------|
| HTTP Client | gorequest | Native net/http |
| Connection Pooling | ❌ | ✅ |
| Context Support | ❌ | ✅ |
| Proxy Support | ❌ | ✅ |
| Batch Sending | ❌ | ✅ |
| Configurable Timeouts | Limited | Full |
| Connection Keep-Alive | ❌ | ✅ |
| Functional Options | ❌ | ✅ |
| Test Coverage | Limited | Comprehensive |
| Go Modules | ✅ | ✅ |

## Installation

```bash
go get github.com/fishfinal/slack-webhook
```

## Quick Start

```go
package main

import (
    "log"
    "github.com/fishfinal/slack-webhook"
)

func main() {
    // Create a new webhook client
    // Note: API is fully compatible with ashwanthkumar/slack-go-webhook
    webhook := slack.NewWebhook("https://hooks.slack.com/services/XXX/YYY/ZZZ")

    // Send a simple message
    payload := slack.Payload{
        Text:     "Hello from Slack Webhook!",
        Username: "MyBot",
        Channel:  "#general",
    }

    if err := webhook.Send(payload); err != nil {
        log.Fatalf("Failed to send message: %v", err)
    }
}
```

## Migration from ashwanthkumar/slack-go-webhook

Migrating from the original library is straightforward as the core API remains compatible:

```go
// Original library
import "github.com/ashwanthkumar/slack-go-webhook"

// New library - just change the import path
import "github.com/fishfinal/slack-webhook"

// Everything else remains the same!
webhook := slack.NewWebhook(url)
payload := slack.Payload{...}
err := webhook.Send(payload)
```

## Advanced Usage

### Configuration Options

```go
// Create with custom configuration
webhook := slack.NewWebhook(
    "https://hooks.slack.com/services/XXX/YYY/ZZZ",
    slack.WithTimeout(15*time.Second),
    slack.WithProxy("http://proxy.example.com:8080"),
    slack.WithConnectionPool(200, 20, 120*time.Second),
    slack.WithMaxIdleConns(100),
    slack.WithMaxIdleConnsPerHost(10),
    slack.WithIdleConnTimeout(90*time.Second),
)
```

### Sending Messages with Attachments

```go
payload := slack.Payload{
    Text:     "Check out this message!",
    Username: "NotificationBot",
    Channel:  "#alerts",
    Attachments: []slack.Attachment{
        {
            Title:  stringPtr("Deployment Status"),
            Text:   stringPtr("Deployment completed successfully!"),
            Color:  stringPtr("#36a64f"), // Green
            Fields: []*slack.Field{
                {Title: "Environment", Value: "Production", Short: true},
                {Title: "Version", Value: "v2.1.0", Short: true},
                {Title: "Deployed By", Value: "@devops", Short: false},
            },
            Footer:     stringPtr("Deployment System"),
            FooterIcon: stringPtr("https://example.com/icon.png"),
            Timestamp:  int64Ptr(time.Now().Unix()),
        },
    },
}

err := webhook.Send(payload)
```

### Sending Messages with Actions (Interactive Buttons)

```go
payload := slack.Payload{
    Text:     "What would you like to do?",
    Channel:  "#general",
    Attachments: []slack.Attachment{
        {
            Text:  stringPtr("Choose an action:"),
            Color: stringPtr("#3AA3E3"),
            Actions: []slack.Action{
                {
                    Type:  "button",
                    Text:  "Approve",
                    Url:   "https://example.com/approve",
                    Style: "primary",
                },
                {
                    Type:  "button",
                    Text:  "Reject",
                    Url:   "https://example.com/reject",
                    Style: "danger",
                },
            },
        },
    },
}

err := webhook.Send(payload)
```

### Batch Sending

```go
payloads := []slack.Payload{
    {Text: "Message 1", Channel: "#general"},
    {Text: "Message 2", Channel: "#general"},
    {Text: "Message 3", Channel: "#general"},
}

errors := webhook.SendBatch(payloads)
for _, err := range errors {
    log.Printf("Batch error: %v", err)
}
```

### Context Support

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// Send with context
if err := webhook.SendWithContext(ctx, payload); err != nil {
    log.Printf("Send failed: %v", err)
}

// Get raw HTTP response with context
resp, err := webhook.SendReceiveResponse(ctx, payload)
if err != nil {
    log.Printf("Request failed: %v", err)
}
defer resp.Body.Close()

// Inspect response
if resp.StatusCode == http.StatusTooManyRequests {
    log.Println("Rate limited!")
}
```

## API Reference

### Types

#### Webhook
The main client structure.

```go
type Webhook struct {
    // Contains unexported fields
}
```

#### Payload
The message structure sent to Slack.

```go
type Payload struct {
    Parse       string       `json:"parse,omitempty"`
    Username    string       `json:"username,omitempty"`
    IconUrl     string       `json:"icon_url,omitempty"`
    IconEmoji   string       `json:"icon_emoji,omitempty"`
    Channel     string       `json:"channel,omitempty"`
    Text        string       `json:"text,omitempty"`
    LinkNames   string       `json:"link_names,omitempty"`
    Attachments []Attachment `json:"attachments,omitempty"`
    UnfurlLinks bool         `json:"unfurl_links,omitempty"`
    UnfurlMedia bool         `json:"unfurl_media,omitempty"`
    Markdown    bool         `json:"mrkdwn,omitempty"`
}
```

#### Attachment
Rich message attachments.

```go
type Attachment struct {
    Fallback     *string   `json:"fallback"`
    Color        *string   `json:"color"`
    PreText      *string   `json:"pretext"`
    AuthorName   *string   `json:"author_name"`
    AuthorLink   *string   `json:"author_link"`
    AuthorIcon   *string   `json:"author_icon"`
    Title        *string   `json:"title"`
    TitleLink    *string   `json:"title_link"`
    Text         *string   `json:"text"`
    ImageUrl     *string   `json:"image_url"`
    Fields       []*Field  `json:"fields"`
    Footer       *string   `json:"footer"`
    FooterIcon   *string   `json:"footer_icon"`
    Timestamp    *int64    `json:"ts"`
    MarkdownIn   *[]string `json:"mrkdwn_in"`
    Actions      []*Action `json:"actions"`
    CallbackID   *string   `json:"callback_id"`
    ThumbnailUrl *string   `json:"thumb_url"`
}
```

### Functions

#### NewWebhook
```go
func NewWebhook(webhookUrl string, options ...Option) *Webhook
```
Creates a new Slack webhook client with optional configuration.

#### Webhook Methods

**Send**
```go
func (w *Webhook) Send(payload Payload) error
```
Sends a message to Slack. Returns error on failure.

**SendWithContext**
```go
func (w *Webhook) SendWithContext(ctx context.Context, payload Payload) error
```
Sends a message with context support for cancellation and timeouts.

**SendReceiveResponse**
```go
func (w *Webhook) SendReceiveResponse(ctx context.Context, payload Payload) (*http.Response, error)
```
Sends a message and returns the raw HTTP response.

**SendBatch**
```go
func (w *Webhook) SendBatch(payloads []Payload) []error
```
Sends multiple messages efficiently using connection pooling.

**Attachment Helpers**
```go
func (attachment *Attachment) AddField(field Field) *Attachment
func (attachment *Attachment) AddAction(action Action) *Attachment
```

### Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithTimeout(duration)` | HTTP client timeout | 10s |
| `WithProxy(url)` | HTTP/HTTPS proxy URL | None |
| `WithConnectionPool(maxIdle, maxPerHost, timeout)` | Connection pool config | 100, 10, 90s |
| `WithMaxIdleConns(n)` | Max idle connections | 100 |
| `WithMaxIdleConnsPerHost(n)` | Max idle connections per host | 10 |
| `WithIdleConnTimeout(duration)` | Idle connection timeout | 90s |

## Error Handling

The package provides detailed error messages:

```go
err := webhook.Send(payload)
if err != nil {
    // Check error type
    if strings.Contains(err.Error(), "context canceled") {
        // Handle cancellation
    } else if strings.Contains(err.Error(), "Status: 429") {
        // Rate limited
    }
    log.Printf("Error: %v", err)
}
```

## Best Practices

### 1. Reuse Webhook Client
Create a single webhook client and reuse it across your application:

```go
var globalWebhook = slack.NewWebhook(
    os.Getenv("SLACK_WEBHOOK_URL"),
    slack.WithTimeout(10*time.Second),
    slack.WithConnectionPool(100, 10, 90*time.Second),
)
```

### 2. Use Context for Timeouts
Always use context for production requests:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

if err := webhook.SendWithContext(ctx, payload); err != nil {
    // Handle error
}
```

### 3. Batch Sending
For multiple messages, use `SendBatch` to leverage connection pooling:

```go
// Instead of:
for _, p := range payloads {
    webhook.Send(p)
}

// Do:
errors := webhook.SendBatch(payloads)
```

## Testing

```bash
# Run tests
go test -v

# Run with coverage
go test -cover

# Run benchmarks
go test -bench=.

# Run specific test
go test -run TestWebhookSuite
```

## Contributing

Contributions are welcome! Here's how you can help:

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please make sure to update tests as appropriate and adhere to the existing coding style.

## License

This project is licensed under the Apache License, Version 2.0 - see the [LICENSE](LICENSE) file for details.

## Credits

- **Original Library**: [ashwanthkumar/slack-go-webhook](https://github.com/ashwanthkumar/slack-go-webhook) by [@ashwanthkumar](https://github.com/ashwanthkumar)
- **Inspiration**: The original library provided the foundation and API design
- **Rewrite & Extensions**: Performance improvements, connection pooling, context support, and additional features
