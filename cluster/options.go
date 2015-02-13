package cluster

import "crypto/tls"

type Options struct {
	TLSConfig       *tls.Config
	OvercommitRatio float64
}
