package token

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/docker/swarm/discovery"
)

// DiscoveryUrl is exported
const DiscoveryURL = "https://discovery-stage.hub.docker.com/v2"

// Discovery is exported
type Discovery struct {
	heartbeat time.Duration
	ttl       time.Duration
	url       string
	token     string
}

func init() {
	Init()
}

// Init is exported
func Init() {
	discovery.Register("token", &Discovery{})
}

// Initialize is exported
func (s *Discovery) Initialize(urltoken string, heartbeat time.Duration, ttl time.Duration) error {
	if i := strings.LastIndex(urltoken, "/"); i != -1 {
		s.url = "https://" + urltoken[:i]
		s.token = urltoken[i+1:]
	} else {
		s.url = DiscoveryURL
		s.token = urltoken
	}

	if s.token == "" {
		return errors.New("token is empty")
	}
	s.heartbeat = heartbeat
	s.ttl = ttl

	return nil
}

// Fetch returns the list of entries for the discovery service at the specified endpoint
func (s *Discovery) fetch() (discovery.Entries, error) {
	resp, err := http.Get(fmt.Sprintf("%s/%s/%s", s.url, "clusters", s.token))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var addrs []string
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&addrs); err != nil {
			return nil, fmt.Errorf("Failed to decode response: %v", err)
		}
	} else {
		return nil, fmt.Errorf("Failed to fetch entries, Discovery service returned %d HTTP status code", resp.StatusCode)
	}

	return discovery.CreateEntries(addrs)
}

// Watch is exported
func (s *Discovery) Watch(stopCh <-chan struct{}) (<-chan discovery.Entries, <-chan error) {
	ch := make(chan discovery.Entries)
	ticker := time.NewTicker(s.heartbeat)
	errCh := make(chan error)

	go func() {
		defer close(ch)
		defer close(errCh)

		// Send the initial entries if available.
		currentEntries, err := s.fetch()
		if err != nil {
			errCh <- err
		} else {
			ch <- currentEntries
		}

		// Periodically send updates.
		for {
			select {
			case <-ticker.C:
				newEntries, err := s.fetch()
				if err != nil {
					errCh <- err
					continue
				}

				// Check if the file has really changed.
				if !newEntries.Equals(currentEntries) {
					ch <- newEntries
				}
				currentEntries = newEntries
			case <-stopCh:
				ticker.Stop()
				return
			}
		}
	}()

	return ch, nil
}

// Register adds a new entry identified by the into the discovery service
func (s *Discovery) Register(addr string) error {
	buf := strings.NewReader(addr)

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/%s/%s", s.url, "clusters", s.token), buf)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	resp.Body.Close()
	return nil
}

func getAuthToken(realm, service, username, password string) (string, error) {
	tokenURL, _ := url.Parse(realm)
	params := url.Values{}
	params.Set("service", service)
	tokenURL.RawQuery = params.Encode()
	req, err := http.NewRequest("GET", tokenURL.String(), nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(username, password)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		rawJSON, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		tokenJSON := struct{ Token string }{}
		if err := json.Unmarshal(rawJSON, &tokenJSON); err != nil {
			return "", err
		}
		return tokenJSON.Token, nil
	}
	if resp.StatusCode == http.StatusUnauthorized {
		rawJSON, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		detailsJSON := struct{ Details string }{}
		if err := json.Unmarshal(rawJSON, &detailsJSON); err != nil {
			return "", err
		}
		if detailsJSON.Details != "" {
			return "", errors.New(detailsJSON.Details)
		}
	}

	return "", errors.New(http.StatusText(resp.StatusCode))
}

// do a POST request to create a token, use authToken if provided
func (s *Discovery) create(authToken, username string) (string, string, error) {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s/%s", s.url, "clusters", username), nil)
	if err != nil {
		return "", "", err
	}
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return "", resp.Header.Get("Www-Authenticate"), nil
	} else if resp.StatusCode != http.StatusOK {
		return "", "", errors.New(http.StatusText(resp.StatusCode))
	}

	token, err := ioutil.ReadAll(resp.Body)
	return string(token), "", err
}

// CreateCluster returns a unique cluster token
func (s *Discovery) CreateCluster(username, password string) (string, error) {
	token, authHeader, err := s.create("", username)
	if err != nil {
		return "", err
	}
	if authHeader != "" {
		var (
			realm   string
			service string
		)
		_, err := fmt.Sscanf(authHeader, "Bearer realm=%q,service=%q", &realm, &service)
		if err != nil {
			return "", err
		}

		authToken, err := getAuthToken(realm, service, username, password)
		if err != nil {
			return "", err
		}
		token, _, err = s.create(authToken, username)
		return token, err
	}
	return token, nil
}

// Destroy do a DELETE request to delete a token, use authToken if provided
func (s *Discovery) destroy(authToken string) (string, error) {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/%s/%s", s.url, "clusters", s.token), nil)
	if err != nil {
		return "", err
	}
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return resp.Header.Get("Www-Authenticate"), nil
	} else if resp.StatusCode != http.StatusOK {
		return "", errors.New(http.StatusText(resp.StatusCode))
	}
	return "", err
}

// DestroyCluster deletes a cluster token
func (s *Discovery) DestroyCluster() (string, error) {
	return s.destroy("")
}

// DestroySecureCluster deletes a secure cluster token
func (s *Discovery) DestroySecureCluster(authHeader, username, password string) error {
	var (
		realm   string
		service string
	)
	_, err := fmt.Sscanf(authHeader, "Bearer realm=%q,service=%q", &realm, &service)
	if err != nil {
		return err
	}

	authToken, err := getAuthToken(realm, service, username, password)
	if err != nil {
		return err
	}
	_, err = s.destroy(authToken)
	return err
}
