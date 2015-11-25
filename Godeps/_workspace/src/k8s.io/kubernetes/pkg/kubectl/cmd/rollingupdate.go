/*
Copyright 2014 The Kubernetes Authors All rights reserved.

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

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/golang/glog"

	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/meta"
	"k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/kubectl"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/kubectl/resource"
	"k8s.io/kubernetes/pkg/util"
)

// RollingUpdateOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type RollingUpdateOptions struct {
	Filenames []string
}

const (
	rollingUpdate_long = `Perform a rolling update of the given ReplicationController.

Replaces the specified replication controller with a new replication controller by updating one pod at a time to use the
new PodTemplate. The new-controller.json must specify the same namespace as the
existing replication controller and overwrite at least one (common) label in its replicaSelector.`
	rollingUpdate_example = `# Update pods of frontend-v1 using new replication controller data in frontend-v2.json.
$ kubectl rolling-update frontend-v1 -f frontend-v2.json

# Update pods of frontend-v1 using JSON data passed into stdin.
$ cat frontend-v2.json | kubectl rolling-update frontend-v1 -f -

# Update the pods of frontend-v1 to frontend-v2 by just changing the image, and switching the
# name of the replication controller.
$ kubectl rolling-update frontend-v1 frontend-v2 --image=image:v2

# Update the pods of frontend by just changing the image, and keeping the old name
$ kubectl rolling-update frontend --image=image:v2
`
)

var (
	updatePeriod, _ = time.ParseDuration("1m0s")
	timeout, _      = time.ParseDuration("5m0s")
	pollInterval, _ = time.ParseDuration("3s")
)

func NewCmdRollingUpdate(f *cmdutil.Factory, out io.Writer) *cobra.Command {
	options := &RollingUpdateOptions{}

	cmd := &cobra.Command{
		Use: "rolling-update OLD_CONTROLLER_NAME ([NEW_CONTROLLER_NAME] --image=NEW_CONTAINER_IMAGE | -f NEW_CONTROLLER_SPEC)",
		// rollingupdate is deprecated.
		Aliases: []string{"rollingupdate"},
		Short:   "Perform a rolling update of the given ReplicationController.",
		Long:    rollingUpdate_long,
		Example: rollingUpdate_example,
		Run: func(cmd *cobra.Command, args []string) {
			err := RunRollingUpdate(f, out, cmd, args, options)
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().Duration("update-period", updatePeriod, `Time to wait between updating pods. Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".`)
	cmd.Flags().Duration("poll-interval", pollInterval, `Time delay between polling for replication controller status after the update. Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".`)
	cmd.Flags().Duration("timeout", timeout, `Max time to wait for a replication controller to update before giving up. Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".`)
	usage := "Filename or URL to file to use to create the new replication controller."
	kubectl.AddJsonFilenameFlag(cmd, &options.Filenames, usage)
	cmd.MarkFlagRequired("filename")
	cmd.Flags().String("image", "", "Image to use for upgrading the replication controller.  Can not be used with --filename/-f")
	cmd.MarkFlagRequired("image")
	cmd.Flags().String("deployment-label-key", "deployment", "The key to use to differentiate between two different controllers, default 'deployment'.  Only relevant when --image is specified, ignored otherwise")
	cmd.Flags().Bool("dry-run", false, "If true, print out the changes that would be made, but don't actually make them.")
	cmd.Flags().Bool("rollback", false, "If true, this is a request to abort an existing rollout that is partially rolled out. It effectively reverses current and next and runs a rollout")
	cmdutil.AddValidateFlags(cmd)
	cmdutil.AddPrinterFlags(cmd)
	return cmd
}

func validateArguments(cmd *cobra.Command, filenames, args []string) (deploymentKey, filename, image, oldName string, err error) {
	deploymentKey = cmdutil.GetFlagString(cmd, "deployment-label-key")
	image = cmdutil.GetFlagString(cmd, "image")
	filename = ""

	if len(deploymentKey) == 0 {
		return "", "", "", "", cmdutil.UsageError(cmd, "--deployment-label-key can not be empty")
	}
	if len(filenames) > 1 {
		return "", "", "", "", cmdutil.UsageError(cmd, "May only specificy a single filename for new controller")
	}
	if len(filenames) > 0 {
		filename = filenames[0]
	}
	if len(filenames) == 0 && len(image) == 0 {
		return "", "", "", "", cmdutil.UsageError(cmd, "Must specify --filename or --image for new controller")
	}
	if len(filenames) != 0 && len(image) != 0 {
		return "", "", "", "", cmdutil.UsageError(cmd, "--filename and --image can not both be specified")
	}
	if len(args) < 1 {
		return "", "", "", "", cmdutil.UsageError(cmd, "Must specify the controller to update")
	}

	return deploymentKey, filename, image, args[0], nil
}

func RunRollingUpdate(f *cmdutil.Factory, out io.Writer, cmd *cobra.Command, args []string, options *RollingUpdateOptions) error {
	if len(os.Args) > 1 && os.Args[1] == "rollingupdate" {
		printDeprecationWarning("rolling-update", "rollingupdate")
	}
	deploymentKey, filename, image, oldName, err := validateArguments(cmd, options.Filenames, args)
	if err != nil {
		return err
	}
	period := cmdutil.GetFlagDuration(cmd, "update-period")
	interval := cmdutil.GetFlagDuration(cmd, "poll-interval")
	timeout := cmdutil.GetFlagDuration(cmd, "timeout")
	dryrun := cmdutil.GetFlagBool(cmd, "dry-run")
	outputFormat := cmdutil.GetFlagString(cmd, "output")

	cmdNamespace, enforceNamespace, err := f.DefaultNamespace()
	if err != nil {
		return err
	}

	client, err := f.Client()
	if err != nil {
		return err
	}

	var newRc *api.ReplicationController
	// fetch rc
	oldRc, err := client.ReplicationControllers(cmdNamespace).Get(oldName)
	if err != nil {
		if !errors.IsNotFound(err) || len(image) == 0 || len(args) > 1 {
			return err
		}
		// We're in the middle of a rename, look for an RC with a source annotation of oldName
		newRc, err := kubectl.FindSourceController(client, cmdNamespace, oldName)
		if err != nil {
			return err
		}
		return kubectl.Rename(client, newRc, oldName)
	}

	var keepOldName bool
	var replicasDefaulted bool

	mapper, typer := f.Object()

	if len(filename) != 0 {
		schema, err := f.Validator(cmdutil.GetFlagBool(cmd, "validate"), cmdutil.GetFlagString(cmd, "schema-cache-dir"))
		if err != nil {
			return err
		}

		request := resource.NewBuilder(mapper, typer, f.ClientMapperForCommand()).
			Schema(schema).
			NamespaceParam(cmdNamespace).DefaultNamespace().
			FilenameParam(enforceNamespace, filename).
			Do()
		obj, err := request.Object()
		if err != nil {
			return err
		}
		var ok bool
		// Handle filename input from stdin. The resource builder always returns an api.List
		// when creating resource(s) from a stream.
		if list, ok := obj.(*api.List); ok {
			if len(list.Items) > 1 {
				return cmdutil.UsageError(cmd, "%s specifies multiple items", filename)
			}
			obj = list.Items[0]
		}
		newRc, ok = obj.(*api.ReplicationController)
		if !ok {
			if _, kind, err := typer.ObjectVersionAndKind(obj); err == nil {
				return cmdutil.UsageError(cmd, "%s contains a %s not a ReplicationController", filename, kind)
			}
			glog.V(4).Infof("Object %#v is not a ReplicationController", obj)
			return cmdutil.UsageError(cmd, "%s does not specify a valid ReplicationController", filename)
		}
		infos, err := request.Infos()
		if err != nil || len(infos) != 1 {
			glog.V(2).Infof("was not able to recover adequate information to discover if .spec.replicas was defaulted")
		} else {
			replicasDefaulted = isReplicasDefaulted(infos[0])
		}
	}
	// If the --image option is specified, we need to create a new rc with at least one different selector
	// than the old rc. This selector is the hash of the rc, which will differ because the new rc has a
	// different image.
	if len(image) != 0 {
		keepOldName = len(args) == 1
		newName := findNewName(args, oldRc)
		if newRc, err = kubectl.LoadExistingNextReplicationController(client, cmdNamespace, newName); err != nil {
			return err
		}
		if newRc != nil {
			fmt.Fprintf(out, "Found existing update in progress (%s), resuming.\n", newRc.Name)
		} else {
			newRc, err = kubectl.CreateNewControllerFromCurrentController(client, cmdNamespace, oldName, newName, image, deploymentKey)
			if err != nil {
				return err
			}
		}
		// Update the existing replication controller with pointers to the 'next' controller
		// and adding the <deploymentKey> label if necessary to distinguish it from the 'next' controller.
		oldHash, err := api.HashObject(oldRc, client.Codec)
		if err != nil {
			return err
		}
		oldRc, err = kubectl.UpdateExistingReplicationController(client, oldRc, cmdNamespace, newRc.Name, deploymentKey, oldHash, out)
		if err != nil {
			return err
		}
	}
	if oldName == newRc.Name {
		return cmdutil.UsageError(cmd, "%s cannot have the same name as the existing ReplicationController %s",
			filename, oldName)
	}

	updater := kubectl.NewRollingUpdater(newRc.Namespace, client)

	// To successfully pull off a rolling update the new and old rc have to differ
	// by at least one selector. Every new pod should have the selector and every
	// old pod should not have the selector.
	var hasLabel bool
	for key, oldValue := range oldRc.Spec.Selector {
		if newValue, ok := newRc.Spec.Selector[key]; ok && newValue != oldValue {
			hasLabel = true
			break
		}
	}
	if !hasLabel {
		return cmdutil.UsageError(cmd, "%s must specify a matching key with non-equal value in Selector for %s",
			filename, oldName)
	}
	// TODO: handle scales during rolling update
	if replicasDefaulted {
		newRc.Spec.Replicas = oldRc.Spec.Replicas
	}
	if dryrun {
		oldRcData := &bytes.Buffer{}
		newRcData := &bytes.Buffer{}
		if outputFormat == "" {
			oldRcData.WriteString(oldRc.Name)
			newRcData.WriteString(newRc.Name)
		} else {
			if err := f.PrintObject(cmd, oldRc, oldRcData); err != nil {
				return err
			}
			if err := f.PrintObject(cmd, newRc, newRcData); err != nil {
				return err
			}
		}
		fmt.Fprintf(out, "Rolling from:\n%s\nTo:\n%s\n", string(oldRcData.Bytes()), string(newRcData.Bytes()))
		return nil
	}
	updateCleanupPolicy := kubectl.DeleteRollingUpdateCleanupPolicy
	if keepOldName {
		updateCleanupPolicy = kubectl.RenameRollingUpdateCleanupPolicy
	}
	config := &kubectl.RollingUpdaterConfig{
		Out:            out,
		OldRc:          oldRc,
		NewRc:          newRc,
		UpdatePeriod:   period,
		Interval:       interval,
		Timeout:        timeout,
		CleanupPolicy:  updateCleanupPolicy,
		MaxUnavailable: util.NewIntOrStringFromInt(0),
		MaxSurge:       util.NewIntOrStringFromInt(1),
	}
	if cmdutil.GetFlagBool(cmd, "rollback") {
		err = kubectl.AbortRollingUpdate(config)
		if err != nil {
			return err
		}
		client.ReplicationControllers(config.NewRc.Namespace).Update(config.NewRc)
	}
	err = updater.Update(config)
	if err != nil {
		return err
	}

	message := "rolling updated"
	if keepOldName {
		newRc.Name = oldName
	} else {
		message = fmt.Sprintf("rolling updated to %q", newRc.Name)
	}
	newRc, err = client.ReplicationControllers(cmdNamespace).Get(newRc.Name)
	if err != nil {
		return err
	}
	if outputFormat != "" {
		return f.PrintObject(cmd, newRc, out)
	}
	_, kind, err := api.Scheme.ObjectVersionAndKind(newRc)
	if err != nil {
		return err
	}
	_, res := meta.KindToResource(kind, false)
	cmdutil.PrintSuccess(mapper, false, out, res, oldName, message)
	return nil
}

func findNewName(args []string, oldRc *api.ReplicationController) string {
	if len(args) >= 2 {
		return args[1]
	}
	if oldRc != nil {
		newName, _ := kubectl.GetNextControllerAnnotation(oldRc)
		return newName
	}
	return ""
}

func isReplicasDefaulted(info *resource.Info) bool {
	if info == nil || info.VersionedObject == nil {
		// was unable to recover versioned info
		return false
	}
	switch info.Mapping.APIVersion {
	case "v1":
		if rc, ok := info.VersionedObject.(*v1.ReplicationController); ok {
			return rc.Spec.Replicas == nil
		}
	}
	return false
}
