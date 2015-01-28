package consulapi

import (
	"time"
)

// SessionEntry represents a session in consul
type SessionEntry struct {
	CreateIndex uint64
	ID          string
	Name        string
	Node        string
	Checks      []string
	LockDelay   time.Duration
	Behavior    string
	TTL         string
}

// Session can be used to query the Session endpoints
type Session struct {
	c *Client
}

// Session returns a handle to the session endpoints
func (c *Client) Session() *Session {
	return &Session{c}
}

// CreateNoChecks is like Create but is used specifically to create
// a session with no associated health checks.
func (s *Session) CreateNoChecks(se *SessionEntry, q *WriteOptions) (string, *WriteMeta, error) {
	body := make(map[string]interface{})
	body["Checks"] = []string{}
	if se != nil {
		if se.Name != "" {
			body["Name"] = se.Name
		}
		if se.Node != "" {
			body["Node"] = se.Node
		}
		if se.LockDelay != 0 {
			body["LockDelay"] = durToMsec(se.LockDelay)
		}
		if se.Behavior != "" {
			body["Behavior"] = se.Behavior
		}
		if se.TTL != "" {
			body["TTL"] = se.TTL
		}
	}
	return s.create(body, q)

}

// Create makes a new session. Providing a session entry can
// customize the session. It can also be nil to use defaults.
func (s *Session) Create(se *SessionEntry, q *WriteOptions) (string, *WriteMeta, error) {
	var obj interface{}
	if se != nil {
		body := make(map[string]interface{})
		obj = body
		if se.Name != "" {
			body["Name"] = se.Name
		}
		if se.Node != "" {
			body["Node"] = se.Node
		}
		if se.LockDelay != 0 {
			body["LockDelay"] = durToMsec(se.LockDelay)
		}
		if len(se.Checks) > 0 {
			body["Checks"] = se.Checks
		}
		if se.Behavior != "" {
			body["Behavior"] = se.Behavior
		}
		if se.TTL != "" {
			body["TTL"] = se.TTL
		}
	}
	return s.create(obj, q)
}

func (s *Session) create(obj interface{}, q *WriteOptions) (string, *WriteMeta, error) {
	r := s.c.newRequest("PUT", "/v1/session/create")
	r.setWriteOptions(q)
	r.obj = obj
	rtt, resp, err := requireOK(s.c.doRequest(r))
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	wm := &WriteMeta{RequestTime: rtt}
	var out struct{ ID string }
	if err := decodeBody(resp, &out); err != nil {
		return "", nil, err
	}
	return out.ID, wm, nil
}

// Destroy invalides a given session
func (s *Session) Destroy(id string, q *WriteOptions) (*WriteMeta, error) {
	r := s.c.newRequest("PUT", "/v1/session/destroy/"+id)
	r.setWriteOptions(q)
	rtt, resp, err := requireOK(s.c.doRequest(r))
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	wm := &WriteMeta{RequestTime: rtt}
	return wm, nil
}

// Renew renews the TTL on a given session
func (s *Session) Renew(id string, q *WriteOptions) (*SessionEntry, *WriteMeta, error) {
	r := s.c.newRequest("PUT", "/v1/session/renew/"+id)
	r.setWriteOptions(q)
	rtt, resp, err := requireOK(s.c.doRequest(r))
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	wm := &WriteMeta{RequestTime: rtt}

	var entries []*SessionEntry
	if err := decodeBody(resp, &entries); err != nil {
		return nil, wm, err
	}

	if len(entries) > 0 {
		return entries[0], wm, nil
	}
	return nil, wm, nil
}

// Info looks up a single session
func (s *Session) Info(id string, q *QueryOptions) (*SessionEntry, *QueryMeta, error) {
	r := s.c.newRequest("GET", "/v1/session/info/"+id)
	r.setQueryOptions(q)
	rtt, resp, err := requireOK(s.c.doRequest(r))
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	qm := &QueryMeta{}
	parseQueryMeta(resp, qm)
	qm.RequestTime = rtt

	var entries []*SessionEntry
	if err := decodeBody(resp, &entries); err != nil {
		return nil, nil, err
	}

	if len(entries) > 0 {
		return entries[0], qm, nil
	}
	return nil, qm, nil
}

// List gets sessions for a node
func (s *Session) Node(node string, q *QueryOptions) ([]*SessionEntry, *QueryMeta, error) {
	r := s.c.newRequest("GET", "/v1/session/node/"+node)
	r.setQueryOptions(q)
	rtt, resp, err := requireOK(s.c.doRequest(r))
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	qm := &QueryMeta{}
	parseQueryMeta(resp, qm)
	qm.RequestTime = rtt

	var entries []*SessionEntry
	if err := decodeBody(resp, &entries); err != nil {
		return nil, nil, err
	}
	return entries, qm, nil
}

// List gets all active sessions
func (s *Session) List(q *QueryOptions) ([]*SessionEntry, *QueryMeta, error) {
	r := s.c.newRequest("GET", "/v1/session/list")
	r.setQueryOptions(q)
	rtt, resp, err := requireOK(s.c.doRequest(r))
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	qm := &QueryMeta{}
	parseQueryMeta(resp, qm)
	qm.RequestTime = rtt

	var entries []*SessionEntry
	if err := decodeBody(resp, &entries); err != nil {
		return nil, nil, err
	}
	return entries, qm, nil
}
