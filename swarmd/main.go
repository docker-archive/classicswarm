package main

import (
	"fmt"
	"log"
	"os"

	"github.com/docker/libcluster"
	"github.com/docker/libcluster/api"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s node1 node2 ...\n", os.Args[0])
		os.Exit(1)
	}
	c := libcluster.NewCluster()
	for _, addr := range os.Args[1:] {
		n := libcluster.NewNode(addr, addr)
		if err := n.Connect(nil); err != nil {
			log.Fatal(err)
		}
		if err := c.AddNode(n); err != nil {
			log.Fatal(err)
		}
	}
	api.ListenAndServe(":4243")
}
