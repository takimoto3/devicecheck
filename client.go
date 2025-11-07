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
	ProductionHost  = "https://api.devicecheck.apple.com"
	DevelopmentHost = "https://api.development.devicecheck.apple.com"

	QueryPath    = "/v1/query_two_bits"
	UpdatePath   = "/v1/update_two_bits"
	ValidatePath = "/v1/validate_device_token"
)

var (
	// 200
	ErrBitStateNotFound = errors.New("bit state not found")
	// 400
	ErrBadDeviceToken        = errors.New("bad device token")
	ErrBadBits               = errors.New("bad bits")
	ErrBadTimestamp          = errors.New("bad timestamp")
	ErrBadAuthorizationToken = errors.New("bad authorization token")
	ErrBadPayload            = errors.New("bad payload")
	// 401
	ErrInvalidAuthorizationToken = errors.New("invalid authorization token")
	ErrAuthorizationTokenExpired = errors.New("authorization token expired")
	// 403
	ErrForbidden = errors.New("forbidden")
	// 405
	ErrMethodNotAllowed = errors.New("method not allowed")
	// 429
	ErrTooManyRequests = errors.New("too many requests")
	// 500
	ErrServerError        = errors.New("server error")
	ErrServiceUnavailable = errors.New("service unavailable")
)

var _ DeviceCheckRequest = &QueryRequest{}
var _ DeviceCheckRequest = &UpdateRequest{}
var _ DeviceCheckRequest = &ValidateRequest{}

type DeviceCheckRequest interface {
	Path() string
}

type QueryRequest struct {
	DeviceToken   string   `json:"device_token"`
	TransactionID string   `json:"transaction_id"`
	TimeStamp     UnixTime `json:"timestamp"`
}

func (r QueryRequest) Path() string { return QueryPath }

type UpdateRequest struct {
	DeviceToken   string   `json:"device_token"`
	TransactionID string   `json:"transaction_id"`
	TimeStamp     UnixTime `json:"timestamp"`
	Bit0          bool     `json:"bit0"`
	Bit1          bool     `json:"bit1"`
}

func (r UpdateRequest) Path() string { return UpdatePath }

type ValidateRequest struct {
	DeviceToken   string   `json:"device_token"`
	TransactionID string   `json:"transaction_id"`
	TimeStamp     UnixTime `json:"timestamp"`
}

func (r ValidateRequest) Path() string { return ValidatePath }

type Result struct {
	Bit0           bool   `json:"bit0"`
	Bit1           bool   `json:"bit1"`
	LastUpdateTime string `json:"last_update_time"`
}

type Response struct {
	Result *Result
}

type Client struct {
	inner     *appleapi.Client
	generator Generator
}

func NewClient(tp token.Provider, gen Generator, opts ...appleapi.Option) (*Client, error) {
	cli, err := appleapi.NewClient(appleapi.DefaultHTTPClientInitializer(), ProductionHost, tp, opts...)
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

func (cli *Client) Do(ctx context.Context, r DeviceCheckRequest) (*Response, error) {
	switch req := r.(type) {
	case *QueryRequest:
		if req.TransactionID == "" {
			req.TransactionID = cli.generator.Generate()
		}
		if req.TimeStamp.Time().IsZero() {
			req.TimeStamp = UnixTime(time.Now().UTC())
		}
	case *UpdateRequest:
		if req.TransactionID == "" {
			req.TransactionID = cli.generator.Generate()
		}
		if req.TimeStamp.Time().IsZero() {
			req.TimeStamp = UnixTime(time.Now().UTC())
		}
	case *ValidateRequest:
		if req.TransactionID == "" {
			req.TransactionID = cli.generator.Generate()
		}
		if req.TimeStamp.Time().IsZero() {
			req.TimeStamp = UnixTime(time.Now().UTC())
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
