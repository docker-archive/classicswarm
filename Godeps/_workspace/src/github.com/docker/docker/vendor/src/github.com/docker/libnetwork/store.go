package libnetwork

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/libnetwork/datastore"
)

func (c *controller) initStores() error {
	c.Lock()
	if c.cfg == nil {
		c.Unlock()
		return nil
	}
	scopeConfigs := c.cfg.Scopes
	c.Unlock()

	for scope, scfg := range scopeConfigs {
		store, err := datastore.NewDataStore(scope, scfg)
		if err != nil {
			return err
		}
		c.Lock()
		c.stores = append(c.stores, store)
		c.Unlock()
	}

	c.startWatch()
	return nil
}

func (c *controller) closeStores() {
	for _, store := range c.getStores() {
		store.Close()
	}
}

func (c *controller) getStore(scope string) datastore.DataStore {
	c.Lock()
	defer c.Unlock()

	for _, store := range c.stores {
		if store.Scope() == scope {
			return store
		}
	}

	return nil
}

func (c *controller) getStores() []datastore.DataStore {
	c.Lock()
	defer c.Unlock()

	return c.stores
}

func (c *controller) getNetworkFromStore(nid string) (*network, error) {
	for _, store := range c.getStores() {
		n := &network{id: nid, ctrlr: c}
		err := store.GetObject(datastore.Key(n.Key()...), n)
		if err != nil && err != datastore.ErrKeyNotFound {
			return nil, fmt.Errorf("could not find network %s: %v", nid, err)
		}

		// Continue searching in the next store if the key is not found in this store
		if err == datastore.ErrKeyNotFound {
			continue
		}

		ec := &endpointCnt{n: n}
		err = store.GetObject(datastore.Key(ec.Key()...), ec)
		if err != nil {
			return nil, fmt.Errorf("could not find endpoint count for network %s: %v", n.Name(), err)
		}

		n.epCnt = ec
		return n, nil
	}

	return nil, fmt.Errorf("network %s not found", nid)
}

func (c *controller) getNetworksFromStore() ([]*network, error) {
	var nl []*network

	for _, store := range c.getStores() {
		kvol, err := store.List(datastore.Key(datastore.NetworkKeyPrefix),
			&network{ctrlr: c})
		if err != nil && err != datastore.ErrKeyNotFound {
			return nil, fmt.Errorf("failed to get networks for scope %s: %v",
				store.Scope(), err)
		}

		// Continue searching in the next store if no keys found in this store
		if err == datastore.ErrKeyNotFound {
			continue
		}

		for _, kvo := range kvol {
			n := kvo.(*network)
			n.ctrlr = c

			ec := &endpointCnt{n: n}
			err = store.GetObject(datastore.Key(ec.Key()...), ec)
			if err != nil {
				return nil, fmt.Errorf("could not find endpoint count key %s for network %s while listing: %v", datastore.Key(ec.Key()...), n.Name(), err)
			}

			n.epCnt = ec
			nl = append(nl, n)
		}
	}

	return nl, nil
}

func (n *network) getEndpointFromStore(eid string) (*endpoint, error) {
	for _, store := range n.ctrlr.getStores() {
		ep := &endpoint{id: eid, network: n}
		err := store.GetObject(datastore.Key(ep.Key()...), ep)
		if err != nil && err != datastore.ErrKeyNotFound {
			return nil, fmt.Errorf("could not find endpoint %s: %v", eid, err)
		}

		// Continue searching in the next store if the key is not found in this store
		if err == datastore.ErrKeyNotFound {
			continue
		}

		return ep, nil
	}

	return nil, fmt.Errorf("endpoint %s not found", eid)
}

func (n *network) getEndpointsFromStore() ([]*endpoint, error) {
	var epl []*endpoint

	tmp := endpoint{network: n}
	for _, store := range n.getController().getStores() {
		kvol, err := store.List(datastore.Key(tmp.KeyPrefix()...), &endpoint{network: n})
		if err != nil && err != datastore.ErrKeyNotFound {
			return nil,
				fmt.Errorf("failed to get endpoints for network %s scope %s: %v",
					n.Name(), store.Scope(), err)
		}

		// Continue searching in the next store if no keys found in this store
		if err == datastore.ErrKeyNotFound {
			continue
		}

		for _, kvo := range kvol {
			ep := kvo.(*endpoint)
			ep.network = n
			epl = append(epl, ep)
		}
	}

	return epl, nil
}

func (c *controller) updateToStore(kvObject datastore.KVObject) error {
	cs := c.getStore(kvObject.DataScope())
	if cs == nil {
		log.Warnf("datastore for scope %s not initialized. kv object %s is not added to the store", kvObject.DataScope(), datastore.Key(kvObject.Key()...))
		return nil
	}

	if err := cs.PutObjectAtomic(kvObject); err != nil {
		if err == datastore.ErrKeyModified {
			return err
		}
		return fmt.Errorf("failed to update store for object type %T: %v", kvObject, err)
	}

	return nil
}

func (c *controller) deleteFromStore(kvObject datastore.KVObject) error {
	cs := c.getStore(kvObject.DataScope())
	if cs == nil {
		log.Debugf("datastore for scope %s not initialized. kv object %s is not deleted from datastore", kvObject.DataScope(), datastore.Key(kvObject.Key()...))
		return nil
	}

retry:
	if err := cs.DeleteObjectAtomic(kvObject); err != nil {
		if err == datastore.ErrKeyModified {
			if err := cs.GetObject(datastore.Key(kvObject.Key()...), kvObject); err != nil {
				return fmt.Errorf("could not update the kvobject to latest when trying to delete: %v", err)
			}
			goto retry
		}
		return err
	}

	return nil
}

type netWatch struct {
	localEps  map[string]*endpoint
	remoteEps map[string]*endpoint
	stopCh    chan struct{}
}

func (c *controller) getLocalEps(nw *netWatch) []*endpoint {
	c.Lock()
	defer c.Unlock()

	var epl []*endpoint
	for _, ep := range nw.localEps {
		epl = append(epl, ep)
	}

	return epl
}

func (c *controller) watchSvcRecord(ep *endpoint) {
	c.watchCh <- ep
}

func (c *controller) unWatchSvcRecord(ep *endpoint) {
	c.unWatchCh <- ep
}

func (c *controller) networkWatchLoop(nw *netWatch, ep *endpoint, ecCh <-chan datastore.KVObject) {
	for {
		select {
		case <-nw.stopCh:
			return
		case o := <-ecCh:
			ec := o.(*endpointCnt)

			epl, err := ec.n.getEndpointsFromStore()
			if err != nil {
				break
			}

			c.Lock()
			var addEp []*endpoint

			delEpMap := make(map[string]*endpoint)
			for k, v := range nw.remoteEps {
				delEpMap[k] = v
			}

			for _, lEp := range epl {
				if _, ok := nw.localEps[lEp.ID()]; ok {
					continue
				}

				if _, ok := nw.remoteEps[lEp.ID()]; ok {
					delete(delEpMap, lEp.ID())
					continue
				}

				nw.remoteEps[lEp.ID()] = lEp
				addEp = append(addEp, lEp)

			}
			c.Unlock()

			for _, lEp := range addEp {
				ep.getNetwork().updateSvcRecord(lEp, c.getLocalEps(nw), true)
			}

			for _, lEp := range delEpMap {
				ep.getNetwork().updateSvcRecord(lEp, c.getLocalEps(nw), false)

			}
		}
	}
}

func (c *controller) processEndpointCreate(nmap map[string]*netWatch, ep *endpoint) {
	c.Lock()
	nw, ok := nmap[ep.getNetwork().ID()]
	c.Unlock()

	if ok {
		// Update the svc db for the local endpoint join right away
		ep.getNetwork().updateSvcRecord(ep, c.getLocalEps(nw), true)

		c.Lock()
		nw.localEps[ep.ID()] = ep
		c.Unlock()
		return
	}

	nw = &netWatch{
		localEps:  make(map[string]*endpoint),
		remoteEps: make(map[string]*endpoint),
	}

	// Update the svc db for the local endpoint join right away
	// Do this before adding this ep to localEps so that we don't
	// try to update this ep's container's svc records
	ep.getNetwork().updateSvcRecord(ep, c.getLocalEps(nw), true)

	c.Lock()
	nw.localEps[ep.ID()] = ep
	nmap[ep.getNetwork().ID()] = nw
	nw.stopCh = make(chan struct{})
	c.Unlock()

	store := c.getStore(ep.getNetwork().DataScope())
	if store == nil {
		return
	}

	if !store.Watchable() {
		return
	}

	ch, err := store.Watch(ep.getNetwork().getEpCnt(), nw.stopCh)
	if err != nil {
		log.Warnf("Error creating watch for network: %v", err)
		return
	}

	go c.networkWatchLoop(nw, ep, ch)
}

func (c *controller) processEndpointDelete(nmap map[string]*netWatch, ep *endpoint) {
	c.Lock()
	nw, ok := nmap[ep.getNetwork().ID()]

	if ok {
		delete(nw.localEps, ep.ID())
		c.Unlock()

		// Update the svc db about local endpoint leave right away
		// Do this after we remove this ep from localEps so that we
		// don't try to remove this svc record from this ep's container.
		ep.getNetwork().updateSvcRecord(ep, c.getLocalEps(nw), false)

		c.Lock()
		if len(nw.localEps) == 0 {
			close(nw.stopCh)
			delete(nmap, ep.getNetwork().ID())
		}
	}
	c.Unlock()
}

func (c *controller) watchLoop(nmap map[string]*netWatch) {
	for {
		select {
		case ep := <-c.watchCh:
			c.processEndpointCreate(nmap, ep)
		case ep := <-c.unWatchCh:
			c.processEndpointDelete(nmap, ep)
		}
	}
}

func (c *controller) startWatch() {
	c.watchCh = make(chan *endpoint)
	c.unWatchCh = make(chan *endpoint)
	nmap := make(map[string]*netWatch)

	go c.watchLoop(nmap)
}
