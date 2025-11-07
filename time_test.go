package devicecheck_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/takimoto3/devicecheck"
)

func TestUnixTime_MarshalJSON(t *testing.T) {
	tm := time.UnixMilli(1730812345678).UTC()
	ut := devicecheck.UnixTime(tm)

	data, err := json.Marshal(ut)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	got := string(data)
	want := "1730812345678"

	if got != want {
		t.Errorf("MarshalJSON = %s; want %s", got, want)
	}
}

func TestUnixTime_UnmarshalJSON(t *testing.T) {
	jsonData := []byte("1730812345678")

	var ut devicecheck.UnixTime
	if err := json.Unmarshal(jsonData, &ut); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	got := time.Time(ut).UTC()
	want := time.UnixMilli(1730812345678).UTC()

	if !got.Equal(want) {
		t.Errorf("UnmarshalJSON = %v; want %v", got, want)
	}
}

func TestUnixTime_Time(t *testing.T) {
	tm := time.Now().UTC().Truncate(time.Millisecond)
	ut := devicecheck.UnixTime(tm)

	got := ut.Time()
	if !got.Equal(tm) {
		t.Errorf("Time() = %v; want %v", got, tm)
	}
}

func TestUnixTime_RoundTrip(t *testing.T) {
	original := devicecheck.UnixTime(time.Now().UTC().Truncate(time.Millisecond))

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded devicecheck.UnixTime
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if !time.Time(original).Equal(time.Time(decoded)) {
		t.Errorf("RoundTrip mismatch: got %v, want %v", decoded, original)
	}
}

func TestUnixTime_String(t *testing.T) {
	tests := map[string]struct {
		t    devicecheck.UnixTime
		want string
	}{
		"no subsecond": {
			t:    devicecheck.UnixTime(time.Date(2025, 11, 5, 12, 34, 56, 0, time.UTC)),
			want: "2025-11-05T12:34:56Z",
		},
		"with milliseconds": {
			t:    devicecheck.UnixTime(time.Date(2025, 11, 5, 12, 34, 56, 20000000, time.UTC)), // 20ms
			want: "2025-11-05T12:34:56.02Z",
		},
		"with nanoseconds": {
			t:    devicecheck.UnixTime(time.Date(2025, 11, 5, 12, 34, 56, 123456789, time.UTC)),
			want: "2025-11-05T12:34:56.123456789Z",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.t.String()
			if got != tt.want {
				t.Errorf("UnixTime.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
