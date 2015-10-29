package authZ

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"net/http/httptest"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

type DefaultImp struct{}

func (*DefaultImp) Init() error {

	return nil
}

func (*DefaultImp) HandleEvent(eventType EVENT_ENUM, w http.ResponseWriter, r *http.Request, next http.Handler, containerId string) {
	switch eventType {
	case CONTAINER_CREATE:
		log.Debug("In create...")
		defer r.Body.Close()
		reqBody, _ := ioutil.ReadAll(r.Body)
		log.Debug("Old body: " + string(reqBody))

		//TODO - Here we just use the token for the tenant name for now
		newBody := bytes.Replace(reqBody, []byte("{"), []byte("{\"Labels\": {\""+tenancyLabel+"\":\""+r.Header.Get(authZTokenHeaderName)+"\"},"), 1)
		log.Debug("New body: " + string(newBody))

		var newQuery string
		if "" != r.URL.Query().Get("name") {
			log.Debug("Postfixing name with Label...")
			newQuery = strings.Replace(r.RequestURI, r.URL.Query().Get("name"), r.URL.Query().Get("name")+r.Header.Get(authZTokenHeaderName), 1)
			log.Debug(newQuery)
		}

		newReq, e1 := modifyRequest(r, bytes.NewReader(newBody), newQuery, "")
		if e1 != nil {
			log.Error(e1)
		}
		next.ServeHTTP(w, newReq)

	case CONTAINER_INSPECT:
		log.Debug("In inspect...")
		rec := httptest.NewRecorder()

		r.URL.Path = strings.Replace(r.URL.Path, mux.Vars(r)["name"], containerId, 1)
		mux.Vars(r)["name"] = containerId
		next.ServeHTTP(rec, r)

		/*POST Swarm*/
		w.WriteHeader(rec.Code)
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}
		newBody := cleanUpLabeling(r, rec)
		w.Write(newBody)

	case CONTAINERS_LIST:
		log.Debug("In list...")
		var v = url.Values{}
		mapS := map[string][]string{"label": []string{tenancyLabel + "=" + r.Header.Get(authZTokenHeaderName)}}
		filterJSON, _ := json.Marshal(mapS)
		v.Set("filters", string(filterJSON))
		var newQuery string
		if strings.Contains(r.URL.RequestURI(), "?") {
			newQuery = r.URL.RequestURI() + "&" + v.Encode()
		} else {
			newQuery = r.URL.RequestURI() + "?" + v.Encode()
		}
		log.Debug("New Query: ", newQuery)

		newReq, e1 := modifyRequest(r, nil, newQuery, containerId)
		if e1 != nil {
			log.Error(e1)
		}
		rec := httptest.NewRecorder()

		//TODO - May decide to overrideSwarms handlers.getContainersJSON - this is Where to do it.
		next.ServeHTTP(rec, newReq)

		/*POST Swarm*/
		w.WriteHeader(rec.Code)
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}

		newBody := cleanUpLabeling(r, rec)

		w.Write(newBody)

	case CONTAINER_OTHERS:
		log.Debug("In others...")
		r.URL.Path = strings.Replace(r.URL.Path, mux.Vars(r)["name"], containerId, 1)
		mux.Vars(r)["name"] = containerId

		next.ServeHTTP(w, r)

		//TODO - hijack and others are the same because we handle no post and no stream manipulation and no handler override yet
	case STREAM_OR_HIJACK:
		log.Debug("In stream/hijack...")
		r.URL.Path = strings.Replace(r.URL.Path, mux.Vars(r)["name"], containerId, 1)
		mux.Vars(r)["name"] = containerId
		next.ServeHTTP(w, r)

	case PASS_AS_IS:
		log.Debug("Forwarding the request AS IS...")
		next.ServeHTTP(w, r)
	case UNAUTHORIZED:
		log.Debug("In UNAUTHORIZED...")
	default:
		log.Debug("In default...")
	}
}
