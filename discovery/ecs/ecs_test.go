package ecs

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/docker/swarm/discovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestInitialize(t *testing.T) {
	d := &Discovery{}
	d.Initialize("cluster", 1000, 0)
	assert.Equal(t, d.cluster, "cluster")
	assert.Equal(t, d.port, "2375")
}

func TestInitializeWithPort(t *testing.T) {
	d := &Discovery{}
	d.Initialize("cluster:2376", 1000, 0)
	assert.Equal(t, d.cluster, "cluster")
	assert.Equal(t, d.port, "2376")
}

func TestNew(t *testing.T) {
	d, err := discovery.New("ecs://cluster", 0, 0)
	assert.NoError(t, err)
	assert.Equal(t, d.(*Discovery).cluster, "cluster")
}

func TestWatch(t *testing.T) {
	ecsMock := &mockECS{}
	ec2Mock := &mockEC2{}

	d := &Discovery{}
	d.Initialize("cluster", 1, 0)
	d.ecs = ecsMock
	d.ec2 = ec2Mock

	ecsMock.On("ListContainerInstances", &ecs.ListContainerInstancesInput{
		Cluster: aws.String("cluster"),
	}).Return(&ecs.ListContainerInstancesOutput{
		ContainerInstanceARNs: []*string{
			aws.String("abcd"),
			aws.String("dcba"),
		},
	}, nil)
	ecsMock.On("DescribeContainerInstances", &ecs.DescribeContainerInstancesInput{
		Cluster: aws.String("cluster"),
		ContainerInstances: []*string{
			aws.String("abcd"),
			aws.String("dcba"),
		},
	}).Return(&ecs.DescribeContainerInstancesOutput{
		ContainerInstances: []*ecs.ContainerInstance{
			{EC2InstanceID: aws.String("i-abcd")},
			{EC2InstanceID: aws.String("i-dcba")},
		},
	}, nil)
	ec2Mock.On("DescribeInstances", &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("instance-id"), Values: []*string{
				aws.String("i-abcd"),
				aws.String("i-dcba"),
			}},
		},
	}).Return(&ec2.DescribeInstancesOutput{
		Reservations: []*ec2.Reservation{
			{
				Instances: []*ec2.Instance{
					{
						PrivateIPAddress: aws.String("10.0.0.1"),
					},
					{
						PrivateIPAddress: aws.String("10.0.0.2"),
					},
				},
			},
		},
	}, nil)

	stopCh := make(chan struct{})
	ch, errCh := d.Watch(stopCh)

	entries := <-ch
	expected := discovery.Entries{
		&discovery.Entry{Host: "10.0.0.1", Port: "2375"},
		&discovery.Entry{Host: "10.0.0.2", Port: "2375"},
	}
	assert.Equal(t, entries, expected)

	// Stop and make sure it closes all channels.
	close(stopCh)
	assert.Nil(t, <-ch)
	assert.Nil(t, <-errCh)

	ecsMock.AssertExpectations(t)
	ec2Mock.AssertExpectations(t)
}

type mockECS struct {
	mock.Mock
}

func (m *mockECS) ListContainerInstances(input *ecs.ListContainerInstancesInput) (*ecs.ListContainerInstancesOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*ecs.ListContainerInstancesOutput), args.Error(1)
}

func (m *mockECS) DescribeContainerInstances(input *ecs.DescribeContainerInstancesInput) (*ecs.DescribeContainerInstancesOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*ecs.DescribeContainerInstancesOutput), args.Error(1)
}

type mockEC2 struct {
	mock.Mock
}

func (m *mockEC2) DescribeInstances(input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*ec2.DescribeInstancesOutput), args.Error(1)
}
