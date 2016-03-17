package cluster

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/samalba/dockerclient"
)

// SwarmLabelNamespace defines the key prefix in all custom labels
const SwarmLabelNamespace = "com.docker.swarm"

// ContainerConfig is exported
// TODO store affinities and constraints in their own fields
type ContainerConfig struct {
	dockerclient.ContainerConfig
}

func parseEnv(e string) (bool, string, string) {
	parts := strings.SplitN(e, ":", 2)
	if len(parts) == 2 {
		return true, parts[0], parts[1]
	}
	return false, "", ""
}

// FIXME: Temporary fix to handle forward/backward compatibility between Docker <1.6 and >=1.7
// ContainerConfig should be handling converting to/from different docker versions
func consolidateResourceFields(c *dockerclient.ContainerConfig) {
	if c.Memory != c.HostConfig.Memory {
		if c.Memory != 0 {
			c.HostConfig.Memory = c.Memory
		} else {
			c.Memory = c.HostConfig.Memory
		}
	}

	if c.MemorySwap != c.HostConfig.MemorySwap {
		if c.MemorySwap != 0 {
			c.HostConfig.MemorySwap = c.MemorySwap
		} else {
			c.MemorySwap = c.HostConfig.MemorySwap
		}
	}

	if c.CpuShares != c.HostConfig.CpuShares {
		if c.CpuShares != 0 {
			c.HostConfig.CpuShares = c.CpuShares
		} else {
			c.CpuShares = c.HostConfig.CpuShares
		}
	}

	if c.Cpuset != c.HostConfig.CpusetCpus {
		if c.Cpuset != "" {
			c.HostConfig.CpusetCpus = c.Cpuset
		} else {
			c.Cpuset = c.HostConfig.CpusetCpus
		}
	}
}

// BuildContainerConfig creates a cluster.ContainerConfig from a dockerclient.ContainerConfig
func BuildContainerConfig(c dockerclient.ContainerConfig) *ContainerConfig {
	var (
		affinities         []string
		constraints        []string
		reschedulePolicies []string
		env                []string
	)

	// only for tests
	if c.Labels == nil {
		c.Labels = make(map[string]string)
	}

	// parse affinities from labels (ex. docker run --label 'com.docker.swarm.affinities=["container==redis","image==nginx"]')
	if labels, ok := c.Labels[SwarmLabelNamespace+".affinities"]; ok {
		json.Unmarshal([]byte(labels), &affinities)
	}

	// parse constraints from labels (ex. docker run --label 'com.docker.swarm.constraints=["region==us-east","storage==ssd"]')
	if labels, ok := c.Labels[SwarmLabelNamespace+".constraints"]; ok {
		json.Unmarshal([]byte(labels), &constraints)
	}

	// parse reschedule policy from labels (ex. docker run --label 'com.docker.swarm.reschedule-policies=on-node-failure')
	if labels, ok := c.Labels[SwarmLabelNamespace+".reschedule-policies"]; ok {
		json.Unmarshal([]byte(labels), &reschedulePolicies)
	}

	// parse affinities/constraints/reschedule policies from env (ex. docker run -e affinity:container==redis -e affinity:image==nginx -e constraint:region==us-east -e constraint:storage==ssd -e reschedule:off)
	for _, e := range c.Env {
		if ok, key, value := parseEnv(e); ok && key == "affinity" {
			affinities = append(affinities, value)
		} else if ok && key == "constraint" {
			constraints = append(constraints, value)
		} else if ok && key == "reschedule" {
			reschedulePolicies = append(reschedulePolicies, value)
		} else {
			env = append(env, e)
		}
	}

	// remove affinities/constraints/reschedule policies from env
	c.Env = env

	// store affinities in labels
	if len(affinities) > 0 {
		if labels, err := json.Marshal(affinities); err == nil {
			c.Labels[SwarmLabelNamespace+".affinities"] = string(labels)
		}
	}

	// store constraints in labels
	if len(constraints) > 0 {
		if labels, err := json.Marshal(constraints); err == nil {
			c.Labels[SwarmLabelNamespace+".constraints"] = string(labels)
		}
	}

	// store reschedule policies in labels
	if len(reschedulePolicies) > 0 {
		if labels, err := json.Marshal(reschedulePolicies); err == nil {
			c.Labels[SwarmLabelNamespace+".reschedule-policies"] = string(labels)
		}
	}

	consolidateResourceFields(&c)

	return &ContainerConfig{c}
}

func (c *ContainerConfig) extractExprs(key string) []string {
	var exprs []string

	if labels, ok := c.Labels[SwarmLabelNamespace+"."+key]; ok {
		json.Unmarshal([]byte(labels), &exprs)
	}

	return exprs
}

// SwarmID extracts the Swarm ID from the Config.
// May return an empty string if not set.
func (c *ContainerConfig) SwarmID() string {
	return c.Labels[SwarmLabelNamespace+".id"]
}

// SetSwarmID sets or overrides the Swarm ID in the Config.
func (c *ContainerConfig) SetSwarmID(id string) {
	c.Labels[SwarmLabelNamespace+".id"] = id
}

// Affinities returns all the affinities from the ContainerConfig
func (c *ContainerConfig) Affinities() []string {
	return c.extractExprs("affinities")
}

// Constraints returns all the constraints from the ContainerConfig
func (c *ContainerConfig) Constraints() []string {
	return c.extractExprs("constraints")
}

// AddAffinity to config
func (c *ContainerConfig) AddAffinity(affinity string) error {
	affinities := c.extractExprs("affinities")
	affinities = append(affinities, affinity)
	labels, err := json.Marshal(affinities)
	if err != nil {
		return err
	}
	c.Labels[SwarmLabelNamespace+".affinities"] = string(labels)
	return nil
}

// RemoveAffinity from config
func (c *ContainerConfig) RemoveAffinity(affinity string) error {
	affinities := []string{}
	for _, a := range c.extractExprs("affinities") {
		if a != affinity {
			affinities = append(affinities, a)
		}
	}
	labels, err := json.Marshal(affinities)
	if err != nil {
		return err
	}
	c.Labels[SwarmLabelNamespace+".affinities"] = string(labels)
	return nil
}

// AddConstraint to config
func (c *ContainerConfig) AddConstraint(constraint string) error {
	constraints := c.extractExprs("constraints")
	constraints = append(constraints, constraint)
	labels, err := json.Marshal(constraints)
	if err != nil {
		return err
	}
	c.Labels[SwarmLabelNamespace+".constraints"] = string(labels)
	return nil
}

// HaveNodeConstraint in config
func (c *ContainerConfig) HaveNodeConstraint() bool {
	constraints := c.extractExprs("constraints")

	for _, constraint := range constraints {
		if strings.HasPrefix(constraint, "node==") && !strings.HasPrefix(constraint, "node==~") {
			return true
		}
	}
	return false
}

// HasReschedulePolicy returns true if the specified policy is part of the config
func (c *ContainerConfig) HasReschedulePolicy(p string) bool {
	for _, reschedulePolicy := range c.extractExprs("reschedule-policies") {
		if reschedulePolicy == p {
			return true
		}
	}
	return false
}

// Validate returns an error if the config isn't valid
func (c *ContainerConfig) Validate() error {
	//TODO: add validation for affinities and constraints
	reschedulePolicies := c.extractExprs("reschedule-policies")
	if len(reschedulePolicies) > 1 {
		return errors.New("too many reschedule policies")
	} else if len(reschedulePolicies) == 1 {
		valid := false
		for _, validReschedulePolicy := range []string{"off", "on-node-failure"} {
			if reschedulePolicies[0] == validReschedulePolicy {
				valid = true
			}
		}
		if !valid {
			return fmt.Errorf("invalid reschedule policy: %s", reschedulePolicies[0])
		}
	}

	return nil
}
