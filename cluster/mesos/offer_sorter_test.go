package mesos

import (
	"sort"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/mesos/mesos-go/mesosproto"
	"github.com/stretchr/testify/assert"
)

func TestOfferSorter(t *testing.T) {
	offers := []*mesosproto.Offer{
		{Id: &mesosproto.OfferID{Value: proto.String("id1")}},
		{Id: &mesosproto.OfferID{Value: proto.String("id3")}},
		{Id: &mesosproto.OfferID{Value: proto.String("id2")}},
	}

	sort.Sort(offerSorter(offers))

	assert.Equal(t, offers[0].Id.GetValue(), "id1")
	assert.Equal(t, offers[1].Id.GetValue(), "id2")
	assert.Equal(t, offers[2].Id.GetValue(), "id3")
}
