package mesos

import (
	"fmt"
	"strings"

	"github.com/docker/docker/pkg/units"
	"github.com/mesos/mesos-go/mesosproto"
)

func formatResource(resource *mesosproto.Resource) string {
	switch resource.GetType() {
	case mesosproto.Value_SCALAR:
		if resource.GetName() == "disk" || resource.GetName() == "mem" {
			return units.BytesSize(resource.GetScalar().GetValue() * 1024 * 1024)
		}
		return fmt.Sprintf("%d", int(resource.GetScalar().GetValue()))

	case mesosproto.Value_RANGES:
		var ranges []string
		for _, r := range resource.GetRanges().GetRange() {
			ranges = append(ranges, fmt.Sprintf("%d-%d", r.GetBegin(), r.GetEnd()))
		}
		return strings.Join(ranges, ", ")
	}
	return "?"
}

func sumScalarResourceValue(offers map[string]*mesosproto.Offer, name string) float64 {
	var value float64
	for _, offer := range offers {
		for _, resource := range offer.Resources {
			if *resource.Name == name {
				value += *resource.Scalar.Value
			}
		}
	}
	return value
}

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
