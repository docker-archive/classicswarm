package cluster

import "crypto/tls"

// Options is exported
type Options struct {
	TLSConfig *tls.Config
	Opts      []string
	Discovery string
	Heartbeat uint64
}
