package devicecheck_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/takimoto3/appleapi-core"
	"github.com/takimoto3/devicecheck"
)

type WantError struct {
	Msg string
}

func (w WantError) Error() string {
	return w.Msg
}

func (w WantError) Equals(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), w.Msg)
}

type mockTokenProvider struct{}

func (m mockTokenProvider) GetToken(now time.Time) (string, error) {
	return "mock-token", nil
}

func TestClient_Do_AllCases_NoMockRequest(t *testing.T) {
	tests := map[string]struct {
		request      devicecheck.DeviceCheckRequest
		wantErr      error
		wantStatus   int
		wantBody     string
		wantBit0     bool
		wantBit1     bool
		wantLastTime string
	}{
		"QueryRequest success": {
			request:      &devicecheck.QueryRequest{DeviceToken: "token", TransactionID: "tx"},
			wantErr:      nil,
			wantStatus:   200,
			wantBody:     `{"bit0":true,"bit1":false,"last_update_time":"2025-11-06T12:00:00Z"}`,
			wantBit0:     true,
			wantBit1:     false,
			wantLastTime: "2025-11-06T12:00:00Z",
		},
		"QueryRequest bit state not found": {
			request:    &devicecheck.QueryRequest{DeviceToken: "token", TransactionID: "tx"},
			wantErr:    devicecheck.ErrBitStateNotFound,
			wantStatus: 200,
			wantBody:   "Bit State Not Found",
		},
		"Bad Device Token": {
			request:    &devicecheck.UpdateRequest{DeviceToken: "bad", TransactionID: "tx"},
			wantErr:    devicecheck.ErrBadDeviceToken,
			wantStatus: 400,
			wantBody:   "Bad Device Token",
		},
		"QueryRequest empty JSON": {
			request:    &devicecheck.QueryRequest{DeviceToken: "token", TransactionID: "tx"},
			wantErr:    nil,
			wantStatus: 200,
			wantBody:   `{}`,
		},
		"Bad Bits": {
			request:    &devicecheck.UpdateRequest{DeviceToken: "token", TransactionID: "tx"},
			wantErr:    devicecheck.ErrBadBits,
			wantStatus: 400,
			wantBody:   "Bad Bits",
		},
		"Bad Timestamp": {
			request:    &devicecheck.UpdateRequest{DeviceToken: "token", TransactionID: "tx"},
			wantErr:    devicecheck.ErrBadTimestamp,
			wantStatus: 400,
			wantBody:   "Bad Timestamp",
		},
		"Bad Authorization Token": {
			request:    &devicecheck.UpdateRequest{DeviceToken: "token", TransactionID: "tx"},
			wantErr:    devicecheck.ErrBadAuthorizationToken,
			wantStatus: 400,
			wantBody:   "Bad Authorization Token",
		},
		"Bad Payload": {
			request:    &devicecheck.UpdateRequest{DeviceToken: "token", TransactionID: "tx"},
			wantErr:    devicecheck.ErrBadPayload,
			wantStatus: 400,
			wantBody:   "Bad Payload",
		},

		"Invalid Authorization Token": {
			request:    &devicecheck.ValidateRequest{DeviceToken: "token", TransactionID: "tx"},
			wantErr:    devicecheck.ErrInvalidAuthorizationToken,
			wantStatus: 401,
			wantBody:   "Invalid Authorization Token",
		},
		"Authorization Token Expired": {
			request:    &devicecheck.ValidateRequest{DeviceToken: "token", TransactionID: "tx"},
			wantErr:    devicecheck.ErrAuthorizationTokenExpired,
			wantStatus: 401,
			wantBody:   "Authorization Token Expired",
		},

		"Forbidden": {
			request:    &devicecheck.QueryRequest{DeviceToken: "token", TransactionID: "tx"},
			wantErr:    devicecheck.ErrForbidden,
			wantStatus: 403,
			wantBody:   "",
		},

		"Method Not Allowed": {
			request:    &devicecheck.QueryRequest{DeviceToken: "token", TransactionID: "tx"},
			wantErr:    devicecheck.ErrMethodNotAllowed,
			wantStatus: 405,
			wantBody:   "",
		},

		"Too Many Requests": {
			request:    &devicecheck.QueryRequest{DeviceToken: "token", TransactionID: "tx"},
			wantErr:    devicecheck.ErrTooManyRequests,
			wantStatus: 429,
			wantBody:   "",
		},

		"Server Error": {
			request:    &devicecheck.QueryRequest{DeviceToken: "token", TransactionID: "tx"},
			wantErr:    devicecheck.ErrServerError,
			wantStatus: 500,
			wantBody:   "Server Error",
		},
		"Service Unavailable": {
			request:    &devicecheck.QueryRequest{DeviceToken: "token", TransactionID: "tx"},
			wantErr:    devicecheck.ErrServiceUnavailable,
			wantStatus: 500,
			wantBody:   "Service Unavailable",
		},
		"Unknown 400 response": {
			request:    &devicecheck.QueryRequest{DeviceToken: "token", TransactionID: "tx"},
			wantErr:    fmt.Errorf("unexpected response: code=400, message=\"Some Unknown Error\""),
			wantStatus: 400,
			wantBody:   "Some Unknown Error",
		},
		"QueryRequest invalid JSON": {
			request:    &devicecheck.QueryRequest{DeviceToken: "token", TransactionID: "tx"},
			wantErr:    WantError{"unexpected end of JSON input"},
			wantStatus: 200,
			wantBody:   `{"bit0":true,"bit1":`, // invalid JSON
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.wantStatus)
				w.Write([]byte(tt.wantBody))
			}))
			defer server.Close()

			cli, _ := devicecheck.NewClient(mockTokenProvider{})
			cli.SetHost(server.URL)

			resp, err := cli.Do(context.Background(), tt.request)

			if tt.wantErr != nil {
				switch want := tt.wantErr.(type) {
				case WantError:
					if !want.Equals(err) {
						t.Fatalf("expected error containing %q, got %v", want.Msg, err)
					}
				default:
					if err == nil || err.Error() != tt.wantErr.Error() {
						t.Fatalf("expected error %v, got %v", tt.wantErr, err)
					}
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if resp.Result != nil {
					if resp.Result.Bit0 != tt.wantBit0 {
						t.Errorf("expected Bit0 %v, got %v", tt.wantBit0, resp.Result.Bit0)
					}
					if resp.Result.Bit1 != tt.wantBit1 {
						t.Errorf("expected Bit1 %v, got %v", tt.wantBit1, resp.Result.Bit1)
					}
					if resp.Result.LastUpdateTime != tt.wantLastTime {
						t.Errorf("expected LastUpdateTime %q, got %q", tt.wantLastTime, resp.Result.LastUpdateTime)
					}
				}
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	client, err := devicecheck.NewClient(mockTokenProvider{})
	if err != nil {
		t.Fatal(err)
	}
	if client.GetHost() != devicecheck.ProductionHost {
		t.Errorf("expected host=%s, got=%s", devicecheck.ProductionHost, client.GetHost())
	}

	client, err = devicecheck.NewClient(mockTokenProvider{}, appleapi.WithDevelopment())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client.GetHost() != devicecheck.DevelopmentHost {
		t.Errorf("expected host=%s, got=%s", devicecheck.DevelopmentHost, client.GetHost())
	}
}
