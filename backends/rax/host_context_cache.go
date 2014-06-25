package rax

import (
	"fmt"
	"os"
	"github.com/docker/libswarm/beam"
)

// A hostbound action is an action that is supplied with an active client to a desired cloud host
type hostbound func(msg *beam.Message, client HttpClient) error

// A HostContext represents an active session with a cloud host
type HostContextCache struct {
	contexts map[string]*HostContext
}

func NewHostContextCache() (contextCache *HostContextCache) {
	return &HostContextCache{
		contexts: make(map[string]*HostContext),
	}
}

func (hcc *HostContextCache) Get(id, name string, rax *raxcloud) (context *HostContext, err error) {
	return NewHostContext(id, name, rax)
}

func (hcc *HostContextCache) GetCached(id, name string, rax *raxcloud) (context *HostContext, err error) {
	var found bool
	if context, found = hcc.contexts[id]; !found {
		if context, err = NewHostContext(id, name, rax); err == nil {
			hcc.contexts[id] = context
		}
	}

	return
}

func (hcc *HostContextCache) Close() {
	for _, context := range hcc.contexts {
		context.Close()
	}
}

type HostContext struct {
	id     string
	name   string
	rax    *raxcloud
	tunnel *os.Process
}

func NewHostContext(id, name string, rax *raxcloud) (hc *HostContext, err error) {
	if tunnelProcess, err := rax.openTunnel(id, 8000, rax.tunnelPort); err != nil {
		return nil, fmt.Errorf("Unable to tunnel to host %s(id:%s): %v", name, id, err)
	} else {
		return &HostContext{
			id:     id,
			name:   name,
			rax:    rax,
			tunnel: tunnelProcess,
		}, nil
	}
}

func (hc *HostContext) Close() {
	if hc.tunnel != nil {
		hc.tunnel.Kill()
		if state, err := hc.tunnel.Wait(); err != nil {
			fmt.Printf("Wait result: state:%v, err:%s\n", state, err)
		}

		hc.tunnel = nil
	}
}

func (hc *HostContext) exec(msg *beam.Message, action hostbound) (err error) {
	if hc.tunnel == nil {
		return fmt.Errorf("Tunnel not open to host %s(id:%s)", hc.name, hc.id)
	}

	if client, err := newDockerHttpClient("tcp://localhost:8000", "v1.10"); err == nil {
		return action(msg, client)
	} else {
		return fmt.Errorf("Unable to init http client: %v", err)
	}
}
