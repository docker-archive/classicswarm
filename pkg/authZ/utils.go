package authZ

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"net/http/httptest"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

//UTILS

func modifyRequest(r *http.Request, body io.Reader, urlStr string, containerId string) (*http.Request, error) {

	rc, ok := body.(io.ReadCloser)
	if !ok && body != nil {
		rc = ioutil.NopCloser(body)
		r.Body = rc
	}
	if urlStr != "" {
		u, err := url.Parse(urlStr)

		if err != nil {
			return nil, err
		}
		r.URL = u
		mux.Vars(r)["name"] = containerId
	}

	return r, nil
}

func cleanUpLabeling(r *http.Request, rec *httptest.ResponseRecorder) []byte {
	newBody := bytes.Replace(rec.Body.Bytes(), []byte(tenancyLabel), []byte(" "), -1)
	//TODO - Here we just use the token for the tenant name for now so we remove it from the data before returning to user.
	newBody = bytes.Replace(newBody, []byte(r.Header.Get(authZTokenHeaderName)), []byte(" "), -1)
	newBody = bytes.Replace(newBody, []byte(",\" \":\" \""), []byte(""), -1)
	log.Debug("Got this new body...", string(newBody))
	return newBody
}
