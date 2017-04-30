package mesos

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/docker/go-units"
	"github.com/docker/swarm/scheduler/filter"
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

func sumRegularScalarResourceValue(offers map[string]*mesosproto.Offer, name string) float64 {
	var value float64
	for _, offer := range offers {
		for _, resource := range offer.Resources {
			if resource.GetRevocable() != nil {
				continue
			}
			if *resource.Name == name {
				value += *resource.Scalar.Value
			}
		}
	}
	return value
}

func sumRevocablecalarResourceValue(offers map[string]*mesosproto.Offer, name string) float64 {
	var value float64
	for _, offer := range offers {
		for _, resource := range offer.Resources {
			if resource.GetRevocable() == nil {
				continue
			}
			if *resource.Name == name {
				value += *resource.Scalar.Value
			}
		}
	}
	return value
}

func requiredResourceType(constraint filter.Expr) (string, error) {
	var (
		pattern        string
		matchRegular   bool
		matchRevocable bool
		err            error
	)

	value := constraint.Value
	op := constraint.Operator

	if value[0] == '/' && value[len(value)-1] == '/' {
		// regexp
		pattern = value[1 : len(value)-1]
	} else {
		// simple match, create the regex for globbing (ex: ub*t* -> ^ub.*t.*$) and match.
		pattern = "^" + strings.Replace(value, "*", ".*", -1) + "$"
	}

	if matchRegular, err = regexp.MatchString(pattern, "regular"); err != nil {
		return "?", err
	}

	if matchRevocable, err = regexp.MatchString(pattern, "revocable"); err != nil {
		return "?", err
	}

	if matchRegular && !matchRevocable {
		switch op {
		case filter.EQ:
			return RegularResourceOnly, nil
		case filter.NOTEQ:
			return RevocableResourceOnly, nil
		}
	} else if matchRevocable && !matchRegular {
		switch op {
		case filter.EQ:
			return RevocableResourceOnly, nil
		case filter.NOTEQ:
			return RegularResourceOnly, nil
		}
	}

	return "?", nil
}
