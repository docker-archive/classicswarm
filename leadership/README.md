# Leadership: Distributed Leader Election for Clustered Environments.

Leadership is a library for a cluster leader election on top of a distributed
Key/Value store.

It's built using Swarm's `pkg/store` and is designed to work across multiple
storage backends.

You can use `leadership` with `Consul`, `etcd` and `Zookeeper`.

```go
// Create a store using pkg/store.
client, err := store.NewStore("consul", []string{"127.0.0.1:8500"}, &store.Config{})
if err != nil {
	panic(err)
}

underwood := leadership.NewCandidate(client, "service/swarm/leader", "underwood")
underwood.RunForElection()

electedCh := underwood.ElectedCh()
for isElected := range electedCh {
	// This loop will run every time there is a change in our leadership
	// status.

	if isElected {
		// We won the election - we are now the leader.
		// Let's do leader stuff, for example, sleep for a while.
		log.Printf("I won the election! I'm now the leader")
		time.Sleep(10 * time.Second)

		// Tired of being a leader? You can resign anytime.
		candidate.Resign()
	} else {
		// We lost the election but are still running for leadership.
		// `elected == false` is the default state and is the first event
		// we'll receive from the channel. After a successful election,
		// this event can get triggered if someone else steals the
		// leadership or if we resign.

		log.Printf("Lost the election, let's try another time")
	}
}
```

It is possible to follow an election in real-time and get notified whenever
there is a change in leadership:
```go
follower := leadership.NewFollower(client, "service/swarm/leader")
follower.FollowElection()
leaderCh := follower.LeaderCh()
for leader := <-leaderCh {
	// Leader is a string containing the value passed to `NewCandidate`.
	log.Printf("%s is now the leader", leader)
}
```

A typical usecase for this is to be able to always send requests to the current
leader.
