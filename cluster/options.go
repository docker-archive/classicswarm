package cluster

import (
	"strconv"
	"strings"
)

// DriverOpts are key=values options
type DriverOpts []string

// String returns a string from the driver options
func (do DriverOpts) String(key string) (string, bool) {
	for _, opt := range do {
		kv := strings.SplitN(opt, "=", 2)
		if kv[0] == key {
			return kv[1], true
		}
	}
	return "", false
}

// Int returns an int64 from the driver options
func (do DriverOpts) Int(key string) (int64, bool) {
	if value, ok := do.String(key); ok {
		v, _ := strconv.ParseInt(value, 0, 64)
		return v, true
	}
	return 0, false
}

// Uint returns an int64 from the driver options
func (do DriverOpts) Uint(key string) (uint64, bool) {
	if value, ok := do.String(key); ok {
		v, _ := strconv.ParseUint(value, 0, 64)
		return v, true
	}
	return 0, false
}

// Float returns a float64 from the driver options
func (do DriverOpts) Float(key string) (float64, bool) {
	if value, ok := do.String(key); ok {
		v, _ := strconv.ParseFloat(value, 64)
		return v, true
	}
	return 0.0, false
}
