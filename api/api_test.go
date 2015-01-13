package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler"
	"github.com/stretchr/testify/assert"
)

func serveRequest(c *cluster.Cluster, s *scheduler.Scheduler, w http.ResponseWriter, req *http.Request) error {
	context := &context{
		cluster:   c,
		scheduler: s,
		version:   "test-version",
	}

	r := createRouter(context, false)
	r.ServeHTTP(w, req)
	return nil
}

func TestGetVersion(t *testing.T) {
	r := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/version", nil)
	assert.NoError(t, err)

	assert.NoError(t, serveRequest(nil, nil, r, req))
	assert.Equal(t, r.Code, http.StatusOK)

	version := struct {
		Version string
	}{}

	json.NewDecoder(r.Body).Decode(&version)
	assert.Equal(t, version.Version, "swarm/test-version")
}
