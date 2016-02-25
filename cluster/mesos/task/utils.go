package task

import (
	"github.com/mesos/mesos-go/mesosproto"
	"github.com/mesos/mesos-go/mesosutil"
)

func getPorts(offer *mesosproto.Offer) (ports []uint64) {
	for _, resource := range offer.Resources {
		if resource.GetName() == "ports" {
			for _, rang := range resource.GetRanges().GetRange() {
				for i := rang.GetBegin(); i <= rang.GetEnd(); i++ {
					ports = append(ports, i)
				}
			}
		}
	}
	return ports
}

func sumReservedScalarResourceValue(offer *mesosproto.Offer, resourceName string, reservedRole string) (resources float64) {
	var value float64

	for _, resource := range offer.Resources {
		if resource.GetName() != resourceName || resource.GetRole() != reservedRole {
			continue
		}
		value += resource.GetScalar().GetValue()
	}

	return value
}

func buildResourcesForTask(specifiedValue float64, offers map[string]*mesosproto.Offer, resourceName string, role string) []*mesosproto.Resource {
	var resources []*mesosproto.Resource
	var reservedResources float64

	// Go throught all offers to sum the reserved resources of the specified role.
	for _, offer := range offers {
		reservedResources += sumReservedScalarResourceValue(offer, resourceName, role)
	}

	// Swarm uses the reserved resource firstly by default. This behaviour could
	// eventually be configurable by end user (use unreserved resources or reserved resources firstly),
	// although we may want to avoid providing that option until we see a valid use case.
	if reservedResources >= specifiedValue {
		resource := mesosutil.NewScalarResource(resourceName, specifiedValue)
		resource.Role = &role
		resources = append(resources, resource)
	} else {
		resource := mesosutil.NewScalarResource(resourceName, reservedResources)
		resource.Role = &role
		resources = append(resources, resource)
		resource = mesosutil.NewScalarResource(resourceName, specifiedValue-reservedResources)
		resources = append(resources, resource)
	}

	return resources
}
