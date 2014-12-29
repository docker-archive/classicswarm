package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler"
	"github.com/docker/swarm/swarmversion"
	"github.com/stretchr/testify/assert"
)

func serveRequest(c *cluster.Cluster, s *scheduler.Scheduler, w http.ResponseWriter, req *http.Request) error {
	context := &context{
		cluster:   c,
		scheduler: s,
		version:   swarmversion.VERSION,
	}

	r, err := createRouter(context, false)
	if err != nil {
		return err
	}
	r.ServeHTTP(w, req)
	return nil
}

func TestGetVersion(t *testing.T) {

	swarmversion.VERSION = "test-version"
	swarmversion.GITCOMMIT = "test-commit"

	r := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/version", nil)
	assert.NoError(t, err)

	assert.NoError(t, serveRequest(nil, nil, r, req))
	assert.Equal(t, r.Code, http.StatusOK)

	version := struct {
		Version   string
		GitCommit string
	}{}

	json.NewDecoder(r.Body).Decode(&version)
	assert.Equal(t, version.Version, "swarm/test-version")
	assert.Equal(t, version.GitCommit, "test-commit")
}
