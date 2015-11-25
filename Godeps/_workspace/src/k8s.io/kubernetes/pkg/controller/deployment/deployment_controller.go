/*
Copyright 2015 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package deployment

import (
	"fmt"
	"math"
	"time"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/experimental"
	"k8s.io/kubernetes/pkg/client/record"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/util"
	deploymentUtil "k8s.io/kubernetes/pkg/util/deployment"
)

type DeploymentController struct {
	client        client.Interface
	expClient     client.ExperimentalInterface
	eventRecorder record.EventRecorder
}

func New(client client.Interface) *DeploymentController {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(client.Events(""))

	return &DeploymentController{
		client:        client,
		expClient:     client.Experimental(),
		eventRecorder: eventBroadcaster.NewRecorder(api.EventSource{Component: "deployment-controller"}),
	}
}

func (d *DeploymentController) Run(syncPeriod time.Duration) {
	go util.Until(func() {
		errs := d.reconcileDeployments()
		for _, err := range errs {
			glog.Errorf("Failed to reconcile: %v", err)
		}
	}, syncPeriod, util.NeverStop)
}

func (d *DeploymentController) reconcileDeployments() []error {
	list, err := d.expClient.Deployments(api.NamespaceAll).List(labels.Everything(), fields.Everything())
	if err != nil {
		return []error{fmt.Errorf("error listing deployments: %v", err)}
	}
	errs := []error{}
	for _, deployment := range list.Items {
		if err := d.reconcileDeployment(&deployment); err != nil {
			errs = append(errs, fmt.Errorf("error in reconciling deployment %s: %v", deployment.Name, err))
		}
	}
	return errs
}

func (d *DeploymentController) reconcileDeployment(deployment *experimental.Deployment) error {
	switch deployment.Spec.Strategy.Type {
	case experimental.RecreateDeploymentStrategyType:
		return d.reconcileRecreateDeployment(*deployment)
	case experimental.RollingUpdateDeploymentStrategyType:
		return d.reconcileRollingUpdateDeployment(*deployment)
	}
	return fmt.Errorf("unexpected deployment strategy type: %s", deployment.Spec.Strategy.Type)
}

func (d *DeploymentController) reconcileRecreateDeployment(deployment experimental.Deployment) error {
	// TODO: implement me.
	return nil
}

func (d *DeploymentController) reconcileRollingUpdateDeployment(deployment experimental.Deployment) error {
	newRC, err := d.getNewRC(deployment)
	if err != nil {
		return err
	}

	oldRCs, err := d.getOldRCs(deployment)
	if err != nil {
		return err
	}

	allRCs := append(oldRCs, newRC)

	// Scale up, if we can.
	scaledUp, err := d.scaleUp(allRCs, newRC, deployment)
	if err != nil {
		return err
	}
	if scaledUp {
		// Update DeploymentStatus
		return d.updateDeploymentStatus(allRCs, newRC, deployment)
	}

	// Scale down, if we can.
	scaledDown, err := d.scaleDown(allRCs, oldRCs, newRC, deployment)
	if err != nil {
		return err
	}
	if scaledDown {
		// Update DeploymentStatus
		return d.updateDeploymentStatus(allRCs, newRC, deployment)
	}
	// TODO: raise an event, neither scaled up nor down.
	return nil
}

func (d *DeploymentController) getOldRCs(deployment experimental.Deployment) ([]*api.ReplicationController, error) {
	return deploymentUtil.GetOldRCs(deployment, d.client)
}

// Returns an RC that matches the intent of the given deployment.
// It creates a new RC if required.
func (d *DeploymentController) getNewRC(deployment experimental.Deployment) (*api.ReplicationController, error) {
	existingNewRC, err := deploymentUtil.GetNewRC(deployment, d.client)
	if err != nil || existingNewRC != nil {
		return existingNewRC, err
	}
	// new RC does not exist, create one.
	namespace := deployment.ObjectMeta.Namespace
	podTemplateSpecHash := deploymentUtil.GetPodTemplateSpecHash(deployment.Spec.Template)
	rcName := fmt.Sprintf("deploymentrc-%d", podTemplateSpecHash)
	newRCTemplate := deploymentUtil.GetNewRCTemplate(deployment)
	newRC := api.ReplicationController{
		ObjectMeta: api.ObjectMeta{
			Name:      rcName,
			Namespace: namespace,
		},
		Spec: api.ReplicationControllerSpec{
			Replicas: 0,
			Selector: newRCTemplate.ObjectMeta.Labels,
			Template: newRCTemplate,
		},
	}
	createdRC, err := d.client.ReplicationControllers(namespace).Create(&newRC)
	if err != nil {
		return nil, fmt.Errorf("error creating replication controller: %v", err)
	}
	return createdRC, nil
}

func (d *DeploymentController) scaleUp(allRCs []*api.ReplicationController, newRC *api.ReplicationController, deployment experimental.Deployment) (bool, error) {
	if newRC.Spec.Replicas == deployment.Spec.Replicas {
		// Scaling up not required.
		return false, nil
	}
	maxSurge, isPercent, err := util.GetIntOrPercentValue(&deployment.Spec.Strategy.RollingUpdate.MaxSurge)
	if err != nil {
		return false, fmt.Errorf("invalid value for MaxSurge: %v", err)
	}
	if isPercent {
		maxSurge = util.GetValueFromPercent(maxSurge, deployment.Spec.Replicas)
	}
	// Find the total number of pods
	currentPodCount := deploymentUtil.GetReplicaCountForRCs(allRCs)
	// Check if we can scale up.
	maxTotalPods := deployment.Spec.Replicas + maxSurge
	if currentPodCount >= maxTotalPods {
		// Cannot scale up.
		return false, nil
	}
	// Scale up.
	scaleUpCount := maxTotalPods - currentPodCount
	scaleUpCount = int(math.Min(float64(scaleUpCount), float64(deployment.Spec.Replicas-newRC.Spec.Replicas)))
	newReplicasCount := newRC.Spec.Replicas + scaleUpCount
	_, err = d.scaleRC(newRC, newReplicasCount)
	if err == nil {
		d.eventRecorder.Eventf(&deployment, "ScalingRC", "Scaled up rc %s to %d", newRC.Name, newReplicasCount)
	}
	return true, err
}

func (d *DeploymentController) scaleDown(allRCs []*api.ReplicationController, oldRCs []*api.ReplicationController, newRC *api.ReplicationController, deployment experimental.Deployment) (bool, error) {
	oldPodsCount := deploymentUtil.GetReplicaCountForRCs(oldRCs)
	if oldPodsCount == 0 {
		// Cant scale down further
		return false, nil
	}
	maxUnavailable, isPercent, err := util.GetIntOrPercentValue(&deployment.Spec.Strategy.RollingUpdate.MaxUnavailable)
	if err != nil {
		return false, fmt.Errorf("invalid value for MaxUnavailable: %v", err)
	}
	if isPercent {
		maxUnavailable = util.GetValueFromPercent(maxUnavailable, deployment.Spec.Replicas)
	}
	// Check if we can scale down.
	minAvailable := deployment.Spec.Replicas - maxUnavailable
	// Find the number of ready pods.
	readyPodCount, err := deploymentUtil.GetAvailablePodsForRCs(d.client, allRCs)
	if err != nil {
		return false, fmt.Errorf("could not find available pods: %v", err)
	}

	if readyPodCount <= minAvailable {
		// Cannot scale down.
		return false, nil
	}
	totalScaleDownCount := readyPodCount - minAvailable
	for _, targetRC := range oldRCs {
		if totalScaleDownCount == 0 {
			// No further scaling required.
			break
		}
		if targetRC.Spec.Replicas == 0 {
			// cannot scale down this RC.
			continue
		}
		// Scale down.
		scaleDownCount := int(math.Min(float64(targetRC.Spec.Replicas), float64(totalScaleDownCount)))
		newReplicasCount := targetRC.Spec.Replicas - scaleDownCount
		_, err = d.scaleRC(targetRC, newReplicasCount)
		if err != nil {
			return false, err
		}
		d.eventRecorder.Eventf(&deployment, "ScalingRC", "Scaled down rc %s to %d", targetRC.Name, newReplicasCount)
		totalScaleDownCount -= scaleDownCount
	}
	return true, err
}

func (d *DeploymentController) updateDeploymentStatus(allRCs []*api.ReplicationController, newRC *api.ReplicationController, deployment experimental.Deployment) error {
	totalReplicas := deploymentUtil.GetReplicaCountForRCs(allRCs)
	updatedReplicas := deploymentUtil.GetReplicaCountForRCs([]*api.ReplicationController{newRC})
	newDeployment := deployment
	// TODO: Reconcile this with API definition. API definition talks about ready pods, while this just computes created pods.
	newDeployment.Status = experimental.DeploymentStatus{
		Replicas:        totalReplicas,
		UpdatedReplicas: updatedReplicas,
	}
	_, err := d.updateDeployment(&newDeployment)
	return err
}

func (d *DeploymentController) scaleRC(rc *api.ReplicationController, newScale int) (*api.ReplicationController, error) {
	// TODO: Using client for now, update to use store when it is ready.
	rc.Spec.Replicas = newScale
	return d.client.ReplicationControllers(rc.ObjectMeta.Namespace).Update(rc)
}

func (d *DeploymentController) updateDeployment(deployment *experimental.Deployment) (*experimental.Deployment, error) {
	// TODO: Using client for now, update to use store when it is ready.
	return d.client.Experimental().Deployments(deployment.ObjectMeta.Namespace).Update(deployment)
}
