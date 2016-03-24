package cluster

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

// DriverOpts are key=values options
type DriverOpts []string

// ErrNoKeyNorEnv is exported
var ErrNoKeyNorEnv = fmt.Errorf("No key in options and no env")

// String returns a string from the driver options
func (do DriverOpts) String(key, env string) (string, error) {
	for _, opt := range do {
		kv := strings.SplitN(opt, "=", 2)
		if kv[0] == key {
			return kv[1], nil
		}
	}
	if env := os.Getenv(env); env != "" {
		return env, nil
	}
	return "", ErrNoKeyNorEnv
}

// Int returns an int64 from the driver options
func (do DriverOpts) Int(key, env string) (int64, error) {
	value, err := do.String(key, env)
	if err != nil {
		return 0, err
	}

	v, err := strconv.ParseInt(value, 0, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}

// Uint returns an int64 from the driver options
func (do DriverOpts) Uint(key, env string) (uint64, error) {
	value, err := do.String(key, env)
	if err != nil {
		return 0, err
	}

	v, err := strconv.ParseUint(value, 0, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}

// Float returns a float64 from the driver options
func (do DriverOpts) Float(key, env string) (float64, error) {
	value, err := do.String(key, env)
	if err != nil {
		return 0.0, err
	}

	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0.0, err
	}
	return v, nil

}

// IP returns an IP address from the driver options
func (do DriverOpts) IP(key, env string) (net.IP, error) {
	value, err := do.String(key, env)
	if err != nil {
		return nil, err
	}
	return net.ParseIP(value), nil
}

// Bool returns a errorean from the driver options
func (do DriverOpts) Bool(key, env string) (bool, error) {
	value, err := do.String(key, env)
	if err != nil {
		return false, err
	}

	b, err := strconv.ParseBool(value)
	if err != nil {
		return false, err
	}
	return b, nil
}
