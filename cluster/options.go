package cluster

import "crypto/tls"

type Options struct {
	TLSConfig       *tls.Config
	OvercommitRatio float64
	Discovery       string
	Heartbeat       int
	EventsHandler   EventHandler
}
