package cluster

import "crypto/tls"

// Options is exported
type Options struct {
	TLS             *TLSConfig
	OvercommitRatio float64
	Discovery       string
	Heartbeat       uint64
}

// TLSConfig is exported
type TLSConfig struct {
	Config *tls.Config
}
