package ecs

import (
	"net"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/docker/swarm/discovery"
)

// Discovery is exported.
type Discovery struct {
	ecs           ecsClient
	ec2           ec2Client
	cluster, port string
	heartbeat     time.Duration
}

func init() {
	Init()
}

// Init is exported.
func Init() {
	discovery.Register("ecs", &Discovery{})
}

func parsePath(path string) (cluster string, port string) {
	cluster, port, err := net.SplitHostPort(path)
	if err != nil {
		return path, "2375"
	}
	return cluster, port
}

// Initialize is exported.
func (s *Discovery) Initialize(path string, heartbeat time.Duration, ttl time.Duration) error {
	config := aws.DefaultConfig
	s.heartbeat = heartbeat
	s.ecs = ecs.New(config)
	s.ec2 = ec2.New(config)
	s.cluster, s.port = parsePath(path)
	return nil
}

func (s *Discovery) fetch() (discovery.Entries, error) {
	var entries discovery.Entries

	list, err := s.ecs.ListContainerInstances(&ecs.ListContainerInstancesInput{
		Cluster: aws.String(s.cluster),
	})
	if err != nil {
		return entries, err
	}

	ecsDesc, err := s.ecs.DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String(s.cluster),
		ContainerInstances: list.ContainerInstanceARNs,
	})
	if err != nil {
		return entries, err
	}

	var instances []*string
	for _, instance := range ecsDesc.ContainerInstances {
		instances = append(instances, instance.EC2InstanceID)
	}

	ec2Desc, err := s.ec2.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("instance-id"),
				Values: instances,
			},
		},
	})
	if err != nil {
		return entries, err
	}

	for _, reservation := range ec2Desc.Reservations {
		for _, instance := range reservation.Instances {
			entries = append(entries, &discovery.Entry{
				Host: *instance.PrivateIPAddress,
				Port: s.port,
			})
		}
	}

	return entries, nil
}

// Watch is exported
func (s *Discovery) Watch(stopCh <-chan struct{}) (<-chan discovery.Entries, <-chan error) {
	ch := make(chan discovery.Entries)
	errCh := make(chan error)
	ticker := time.NewTicker(s.heartbeat)

	go func() {
		defer close(errCh)
		defer close(ch)

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

	return ch, errCh
}

// Register is exported
func (s *Discovery) Register(addr string) error {
	return discovery.ErrNotImplemented
}

type ecsClient interface {
	ListContainerInstances(*ecs.ListContainerInstancesInput) (*ecs.ListContainerInstancesOutput, error)
	DescribeContainerInstances(*ecs.DescribeContainerInstancesInput) (*ecs.DescribeContainerInstancesOutput, error)
}

type ec2Client interface {
	DescribeInstances(*ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error)
}
