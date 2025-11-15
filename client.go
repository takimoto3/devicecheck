// Package devicecheck provides a client for the Apple DeviceCheck API.
package devicecheck

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/takimoto3/appleapi-core"
	"github.com/takimoto3/appleapi-core/token"
)

const (
	// ProductionHost is the base URL for the production DeviceCheck API environment.
	ProductionHost = "https://api.devicecheck.apple.com"
	// DevelopmentHost is the base URL for the development DeviceCheck API environment.
	DevelopmentHost = "https://api.development.devicecheck.apple.com"

	// QueryPath is the API endpoint for querying two bits.
	QueryPath = "/v1/query_two_bits"
	// UpdatePath is the API endpoint for updating two bits.
	UpdatePath = "/v1/update_two_bits"
	// ValidatePath is the API endpoint for validating a device token.
	ValidatePath = "/v1/validate_device_token"
)

var (
	// ErrBitStateNotFound indicates that the bit state was not found (HTTP 200 with specific message).
	ErrBitStateNotFound = errors.New("bit state not found")
	// ErrBadDeviceToken indicates a bad device token (HTTP 400).
	ErrBadDeviceToken = errors.New("bad device token")
	// ErrBadBits indicates bad bits in the request (HTTP 400).
	ErrBadBits = errors.New("bad bits")
	// ErrBadTimestamp indicates a bad timestamp in the request (HTTP 400).
	ErrBadTimestamp = errors.New("bad timestamp")
	// ErrBadAuthorizationToken indicates a bad authorization token (HTTP 400).
	ErrBadAuthorizationToken = errors.New("bad authorization token")
	// ErrBadPayload indicates a bad request payload (HTTP 400).
	ErrBadPayload = errors.New("bad payload")
	// ErrInvalidAuthorizationToken indicates an invalid authorization token (HTTP 401).
	ErrInvalidAuthorizationToken = errors.New("invalid authorization token")
	// ErrAuthorizationTokenExpired indicates an expired authorization token (HTTP 401).
	ErrAuthorizationTokenExpired = errors.New("authorization token expired")
	// ErrForbidden indicates that the request is forbidden (HTTP 403).
	ErrForbidden = errors.New("forbidden")
	// ErrMethodNotAllowed indicates that the HTTP method is not allowed (HTTP 405).
	ErrMethodNotAllowed = errors.New("method not allowed")
	// ErrTooManyRequests indicates too many requests (HTTP 429).
	ErrTooManyRequests = errors.New("too many requests")
	// ErrServerError indicates a server-side error (HTTP 500).
	ErrServerError = errors.New("server error")
	// ErrServiceUnavailable indicates that the service is unavailable (HTTP 500).
	ErrServiceUnavailable = errors.New("service unavailable")
)

var _ DeviceCheckRequest = &QueryRequest{}
var _ DeviceCheckRequest = &UpdateRequest{}
var _ DeviceCheckRequest = &ValidateRequest{}

// DeviceCheckRequest is an interface for all DeviceCheck API request types.
type DeviceCheckRequest interface {
	Path() string
}

// QueryRequest represents a request to query the two bits for a device.
type QueryRequest struct {
	DeviceToken   string            `json:"device_token"`
	TransactionID string            `json:"transaction_id"`
	TimeStamp     appleapi.UnixTime `json:"timestamp"`
}

// Path returns the API path for the QueryRequest.
func (r QueryRequest) Path() string { return QueryPath }

// UpdateRequest represents a request to update the two bits for a device.
type UpdateRequest struct {
	DeviceToken   string            `json:"device_token"`
	TransactionID string            `json:"transaction_id"`
	TimeStamp     appleapi.UnixTime `json:"timestamp"`
	Bit0          bool              `json:"bit0"`
	Bit1          bool              `json:"bit1"`
}

// Path returns the API path for the UpdateRequest.
func (r UpdateRequest) Path() string { return UpdatePath }

// ValidateRequest represents a request to validate a device token.
type ValidateRequest struct {
	DeviceToken   string            `json:"device_token"`
	TransactionID string            `json:"transaction_id"`
	TimeStamp     appleapi.UnixTime `json:"timestamp"`
}

// Path returns the API path for the ValidateRequest.
func (r ValidateRequest) Path() string { return ValidatePath }

// Result represents the result of a DeviceCheck query, containing the state of the two bits and the last update time.
type Result struct {
	Bit0           bool   `json:"bit0"`
	Bit1           bool   `json:"bit1"`
	LastUpdateTime string `json:"last_update_time"`
}

// Response represents the full response from a DeviceCheck API call.
type Response struct {
	Result *Result
}

// Client is the DeviceCheck API client.
type Client struct {
	inner     *appleapi.Client
	generator Generator
}

// NewClient creates a new DeviceCheck client with a default HTTP client initializer.
func NewClient(tp token.Provider, gen Generator, opts ...appleapi.Option) (*Client, error) {
	return NewClientFromInitializer(appleapi.DefaultHTTPClientInitializer(), tp, gen, opts...)
}

// NewClientFromInitializer creates a new DeviceCheck client with a custom HTTP client initializer.
func NewClientFromInitializer(initializer appleapi.HTTPClientInitializer, tp token.Provider, gen Generator, opts ...appleapi.Option) (*Client, error) {
	cli, err := appleapi.NewClient(initializer, ProductionHost, tp, opts...)
	if err != nil {
		return nil, err
	}
	if cli.Development {
		cli.Host = DevelopmentHost
	}

	if gen == nil {
		gen = UUIDGenerator{}
	}

	return &Client{inner: cli, generator: gen}, nil
}

// Do sends a DeviceCheck API request and returns the response.
// It automatically populates TransactionID and TimeStamp if they are empty in the request.
func (cli *Client) Do(ctx context.Context, r DeviceCheckRequest) (*Response, error) {
	switch req := r.(type) {
	case *QueryRequest:
		if req.TransactionID == "" {
			req.TransactionID = cli.generator.Generate()
		}
		if req.TimeStamp.Time().IsZero() {
			req.TimeStamp = appleapi.UnixTime(time.Now().UTC())
		}
	case *UpdateRequest:
		if req.TransactionID == "" {
			req.TransactionID = cli.generator.Generate()
		}
		if req.TimeStamp.Time().IsZero() {
			req.TimeStamp = appleapi.UnixTime(time.Now().UTC())
		}
	case *ValidateRequest:
		if req.TransactionID == "" {
			req.TransactionID = cli.generator.Generate()
		}
		if req.TimeStamp.Time().IsZero() {
			req.TimeStamp = appleapi.UnixTime(time.Now().UTC())
		}
	}

	url := cli.inner.Host + r.Path()

	data, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := cli.inner.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	return cli.handleResponse(resp)
}

// handleResponse processes the HTTP response from the DeviceCheck API.
func (cli *Client) handleResponse(resp *http.Response) (*Response, error) {
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		cli.inner.Logger.Error("failed to read response body",
			"status", resp.StatusCode,
			"error", err,
		)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	msg := strings.TrimSpace(string(data))

	cli.inner.Logger.Debug("DeviceCheck response received",
		"status", resp.StatusCode,
		"message", msg,
	)

	switch resp.StatusCode {
	case 200:
		if len(msg) == 0 {
			return &Response{&Result{}}, nil
		}
		if strings.Contains(msg, "Bit State Not Found") {
			cli.inner.Logger.Info("bit state not found", "status", resp.StatusCode)
			return &Response{&Result{}}, ErrBitStateNotFound
		}

		var result Result
		if err := json.Unmarshal(data, &result); err != nil {
			cli.inner.Logger.Warn("failed to decode response JSON", "status", resp.StatusCode, "message", msg, "error", err)
			return nil, fmt.Errorf("failed to decode JSON: %w", err)
		}
		return &Response{Result: &result}, nil
	case 400:
		cli.inner.Logger.Warn("bad request", "status", resp.StatusCode, "message", msg)
		switch msg {
		case "Bad Device Token":
			return nil, ErrBadDeviceToken
		case "Bad Bits":
			return nil, ErrBadBits
		case "Bad Timestamp":
			return nil, ErrBadTimestamp
		case "Bad Authorization Token":
			return nil, ErrBadAuthorizationToken
		case "Bad Payload":
			return nil, ErrBadPayload
		}

	case 401:
		cli.inner.Logger.Warn("unauthorized", "status", resp.StatusCode, "message", msg)
		switch msg {
		case "Invalid Authorization Token":
			return nil, ErrInvalidAuthorizationToken
		case "Authorization Token Expired":
			return nil, ErrAuthorizationTokenExpired
		}

	case 403:
		cli.inner.Logger.Error("forbidden", "status", resp.StatusCode)
		return nil, ErrForbidden

	case 405:
		cli.inner.Logger.Error("method not allowed", "status", resp.StatusCode)
		return nil, ErrMethodNotAllowed

	case 429:
		cli.inner.Logger.Warn("too many requests", "status", resp.StatusCode)
		return nil, ErrTooManyRequests

	case 500:
		cli.inner.Logger.Error("server error", "status", resp.StatusCode, "message", msg)
		switch msg {
		case "Server Error":
			return nil, ErrServerError
		case "Service Unavailable":
			return nil, ErrServiceUnavailable
		}
	}

	// Fallback for all unhandled status codes or unexpected messages.
	cli.inner.Logger.Error("unexpected response",
		"status", resp.StatusCode,
		"message", msg,
	)
	return nil, fmt.Errorf("unexpected response: code=%d, message=%q", resp.StatusCode, msg)
}
