// Package devicecheck provides a client for the Apple DeviceCheck API.
package devicecheck

import (
	"strconv"
	"time"
)

// UnixTime represents a time in milliseconds since Unix epoch (UTC).
type UnixTime time.Time

// MarshalJSON implements the json.Marshaler interface for UnixTime.
// It marshals the time into a Unix timestamp in milliseconds.
func (t UnixTime) MarshalJSON() ([]byte, error) {
	millisec := time.Time(t).UTC().UnixMilli()
	return strconv.AppendInt(nil, millisec, 10), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface for UnixTime.
// It unmarshals a Unix timestamp in milliseconds into a UnixTime.
func (t *UnixTime) UnmarshalJSON(data []byte) error {
	millisec, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return err
	}
	*t = UnixTime(time.UnixMilli(millisec).UTC())
	return nil
}

// Time returns the UnixTime as a standard time.Time.
func (t UnixTime) Time() time.Time {
	return time.Time(t)
}

// String returns the UnixTime as a formatted string (RFC3339Nano).
func (t UnixTime) String() string {
	return time.Time(t).Format(time.RFC3339Nano)
}
