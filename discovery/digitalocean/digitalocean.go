package digitalocean

import (
	"time"
	"os"
	"strconv"
	"github.com/docker/swarm/discovery"
	"github.com/digitalocean/godo"
	"code.google.com/p/goauth2/oauth"
)

type DigitalOceanService struct {
	heartbeat int
	token string
	port string
	network string
}

const DefaultDockerPort = "2375"

func init() {
	discovery.Register("digitalocean", &DigitalOceanService{})
}

func (s *DigitalOceanService) Initialize(token string, heartbeat int) error {
	s.token = token
	s.heartbeat = heartbeat

	s.port = os.Getenv("SWARM_DISCOVERY_DIGITALOCEAN_PORT")
	if _, err := strconv.Atoi(s.port); err != nil {
		s.port = DefaultDockerPort
	}

	s.network = os.Getenv("SWARM_DISCOVERY_DIGITALOCEAN_NETWORK")
	switch s.network {
	    case "private": break
	    default: s.network = "public" //default digitalocean network
    }
	return nil
}

func dropletList(client *godo.Client) ([]godo.Droplet, error) {
    // create a list to hold our droplets
    list := []godo.Droplet{}

    // create options. initially, these will be blank
    opt := &godo.ListOptions{}
    for {
        droplets, resp, err := client.Droplets.List(opt)
        if err != nil {
            return nil, err
        }

        // append the current page's droplets to our list
        for _, d := range droplets {
            list = append(list, d)
        }

        // if we are at the last page, break out the for loop
        if resp.Links == nil || resp.Links.IsLastPage() {
            break
        }

        page, err := resp.Links.CurrentPage()
        if err != nil {
            return nil, err
        }

        // set the page we want for the next request
        opt.Page = page + 1
    }
    return list, nil
}

func parseDigitalOceanResponse(droplets []godo.Droplet, s *DigitalOceanService) []string {
	var result []string
	for _, d := range droplets {
		for _, net := range d.Networks.V4 {
			if net.Type == s.network {
				result = append(result, net.IPAddress + ":" + s.port)
			}
		}
	}
	return result
}

func (s *DigitalOceanService) Fetch() ([]*discovery.Entry, error) {
	t := &oauth.Transport{
	    Token: &oauth.Token{AccessToken: s.token},
	}
	client := godo.NewClient(t.Client())

	list, err := dropletList(client)
	if err != nil {
	    return nil, err
	}
	return discovery.CreateEntries(parseDigitalOceanResponse(list, s))
}

func (s *DigitalOceanService) Watch(callback discovery.WatchCallback) {
	for _ = range time.Tick(time.Duration(s.heartbeat) * time.Second) {
		entries, err := s.Fetch()
		if err == nil {
			callback(entries)
		}
	}
}

func (s *DigitalOceanService) Register(addr string) error {
	return discovery.ErrNotImplemented
}