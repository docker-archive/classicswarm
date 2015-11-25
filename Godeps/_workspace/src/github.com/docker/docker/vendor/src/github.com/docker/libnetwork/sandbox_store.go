package libnetwork

import (
	"container/heap"
	"encoding/json"

	"github.com/Sirupsen/logrus"
	"github.com/docker/libnetwork/datastore"
	"github.com/docker/libnetwork/osl"
)

const (
	sandboxPrefix = "sandbox"
)

type epState struct {
	Eid string
	Nid string
}

type sbState struct {
	ID       string
	Cid      string
	c        *controller
	dbIndex  uint64
	dbExists bool
	Eps      []epState
}

func (sbs *sbState) Key() []string {
	return []string{sandboxPrefix, sbs.ID}
}

func (sbs *sbState) KeyPrefix() []string {
	return []string{sandboxPrefix}
}

func (sbs *sbState) Value() []byte {
	b, err := json.Marshal(sbs)
	if err != nil {
		return nil
	}
	return b
}

func (sbs *sbState) SetValue(value []byte) error {
	return json.Unmarshal(value, sbs)
}

func (sbs *sbState) Index() uint64 {
	sbi, err := sbs.c.SandboxByID(sbs.ID)
	if err != nil {
		return sbs.dbIndex
	}

	sb := sbi.(*sandbox)
	maxIndex := sb.dbIndex
	if sbs.dbIndex > maxIndex {
		maxIndex = sbs.dbIndex
	}

	return maxIndex
}

func (sbs *sbState) SetIndex(index uint64) {
	sbs.dbIndex = index
	sbs.dbExists = true

	sbi, err := sbs.c.SandboxByID(sbs.ID)
	if err != nil {
		return
	}

	sb := sbi.(*sandbox)
	sb.dbIndex = index
	sb.dbExists = true
}

func (sbs *sbState) Exists() bool {
	if sbs.dbExists {
		return sbs.dbExists
	}

	sbi, err := sbs.c.SandboxByID(sbs.ID)
	if err != nil {
		return false
	}

	sb := sbi.(*sandbox)
	return sb.dbExists
}

func (sbs *sbState) Skip() bool {
	return false
}

func (sbs *sbState) New() datastore.KVObject {
	return &sbState{c: sbs.c}
}

func (sbs *sbState) CopyTo(o datastore.KVObject) error {
	dstSbs := o.(*sbState)
	dstSbs.c = sbs.c
	dstSbs.ID = sbs.ID
	dstSbs.Cid = sbs.Cid
	dstSbs.dbIndex = sbs.dbIndex
	dstSbs.dbExists = sbs.dbExists

	for _, eps := range sbs.Eps {
		dstSbs.Eps = append(dstSbs.Eps, eps)
	}

	return nil
}

func (sbs *sbState) DataScope() string {
	return datastore.LocalScope
}

func (sb *sandbox) storeUpdate() error {
	sbs := &sbState{
		c:  sb.controller,
		ID: sb.id,
	}

	for _, ep := range sb.getConnectedEndpoints() {
		eps := epState{
			Nid: ep.getNetwork().ID(),
			Eid: ep.ID(),
		}

		sbs.Eps = append(sbs.Eps, eps)
	}

	return sb.controller.updateToStore(sbs)
}

func (sb *sandbox) storeDelete() error {
	sbs := &sbState{
		c:        sb.controller,
		ID:       sb.id,
		Cid:      sb.containerID,
		dbIndex:  sb.dbIndex,
		dbExists: sb.dbExists,
	}

	return sb.controller.deleteFromStore(sbs)
}

func (c *controller) sandboxCleanup() {
	store := c.getStore(datastore.LocalScope)
	if store == nil {
		logrus.Errorf("Could not find local scope store while trying to cleanup sandboxes")
		return
	}

	kvol, err := store.List(datastore.Key(sandboxPrefix), &sbState{c: c})
	if err != nil && err != datastore.ErrKeyNotFound {
		logrus.Errorf("failed to get sandboxes for scope %s: %v", store.Scope(), err)
		return
	}

	// It's normal for no sandboxes to be found. Just bail out.
	if err == datastore.ErrKeyNotFound {
		return
	}

	for _, kvo := range kvol {
		sbs := kvo.(*sbState)

		sb := &sandbox{
			id:          sbs.ID,
			controller:  sbs.c,
			containerID: sbs.Cid,
			endpoints:   epHeap{},
			epPriority:  map[string]int{},
			dbIndex:     sbs.dbIndex,
			dbExists:    true,
		}

		sb.osSbox, err = osl.NewSandbox(sb.Key(), true)
		if err != nil {
			logrus.Errorf("failed to create new osl sandbox while trying to build sandbox for cleanup: %v", err)
			continue
		}

		for _, eps := range sbs.Eps {
			n, err := c.getNetworkFromStore(eps.Nid)
			if err != nil {
				logrus.Errorf("getNetworkFromStore for nid %s failed while trying to build sandbox for cleanup: %v", eps.Nid, err)
				continue
			}

			ep, err := n.getEndpointFromStore(eps.Eid)
			if err != nil {
				logrus.Errorf("getEndpointFromStore for eid %s failed while trying to build sandbox for cleanup: %v", eps.Eid, err)
				continue
			}

			heap.Push(&sb.endpoints, ep)
		}

		c.Lock()
		c.sandboxes[sb.id] = sb
		c.Unlock()

		if err := sb.Delete(); err != nil {
			logrus.Errorf("failed to delete sandbox %s while trying to cleanup: %v", sb.id, err)
		}
	}
}
