Discovery
=========

Contributing a new discovery backend is easy,
simply implements this interface:

```go
type DiscoveryService interface {
     Fetch() ([]string, error)
     Watch(int) <-chan time.Time
     Register(string) error
}
```

######Fetch
returns the list of all the nodes from the discovery

######Watch
triggers when you need to update (`Fetch`) the list of nodes,
it can happen either via un timer (like `token`) or use
backend specific features (like `etcd`)

######Register
add a new node to the discovery
