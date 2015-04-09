package cluster

import "crypto/tls"

// Options is exported
type Options struct {
	TLSConfig       *tls.Config
	OvercommitRatio float64
	Discovery       string
	Heartbeat       uint64
}
