package devicecheck

import (
	"strconv"
	"time"
)

// UnixTime represents a time in milliseconds since Unix epoch (UTC).
type UnixTime time.Time

func (t UnixTime) MarshalJSON() ([]byte, error) {
	millisec := time.Time(t).UTC().UnixMilli()
	return strconv.AppendInt(nil, millisec, 10), nil
}

func (t *UnixTime) UnmarshalJSON(data []byte) error {
	millisec, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return err
	}
	*t = UnixTime(time.UnixMilli(millisec).UTC())
	return nil
}

func (t UnixTime) Time() time.Time {
	return time.Time(t)
}

func (t UnixTime) String() string {
	return time.Time(t).Format(time.RFC3339Nano)
}
