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
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/kubectl"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/kubectl/resource"
	"k8s.io/kubernetes/pkg/util/yaml"
)

// PatchOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type PatchOptions struct {
	Filenames []string
}

const (
	patch_long = `Update field(s) of a resource using strategic merge patch

JSON and YAML formats are accepted.

Please refer to the models in https://htmlpreview.github.io/?https://github.com/kubernetes/kubernetes/HEAD/docs/api-reference/definitions.html to find if a field is mutable.`
	patch_example = `
# Partially update a node using strategic merge patch
kubectl patch node k8s-node-1 -p '{"spec":{"unschedulable":true}}'

# Partially update a node identified by the type and name specified in "node.json" using strategic merge patch
kubectl patch -f node.json -p '{"spec":{"unschedulable":true}}'

# Update a container's image; spec.containers[*].name is required because it's a merge key
kubectl patch pod valid-pod -p '{"spec":{"containers":[{"name":"kubernetes-serve-hostname","image":"new image"}]}}'`
)

func NewCmdPatch(f *cmdutil.Factory, out io.Writer) *cobra.Command {
	options := &PatchOptions{}

	cmd := &cobra.Command{
		Use:     "patch (-f FILENAME | TYPE NAME) -p PATCH",
		Short:   "Update field(s) of a resource by stdin.",
		Long:    patch_long,
		Example: patch_example,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(cmdutil.ValidateOutputArgs(cmd))
			shortOutput := cmdutil.GetFlagString(cmd, "output") == "name"
			err := RunPatch(f, out, cmd, args, shortOutput, options)
			cmdutil.CheckErr(err)
		},
	}
	cmd.Flags().StringP("patch", "p", "", "The patch to be applied to the resource JSON file.")
	cmd.MarkFlagRequired("patch")
	cmdutil.AddOutputFlagsForMutation(cmd)

	usage := "Filename, directory, or URL to a file identifying the resource to update"
	kubectl.AddJsonFilenameFlag(cmd, &options.Filenames, usage)
	return cmd
}

func RunPatch(f *cmdutil.Factory, out io.Writer, cmd *cobra.Command, args []string, shortOutput bool, options *PatchOptions) error {
	cmdNamespace, enforceNamespace, err := f.DefaultNamespace()
	if err != nil {
		return err
	}

	patch := cmdutil.GetFlagString(cmd, "patch")
	if len(patch) == 0 {
		return cmdutil.UsageError(cmd, "Must specify -p to patch")
	}
	patchBytes, err := yaml.ToJSON([]byte(patch))
	if err != nil {
		return fmt.Errorf("unable to parse %q: %v", patch, err)
	}

	mapper, typer := f.Object()
	r := resource.NewBuilder(mapper, typer, f.ClientMapperForCommand()).
		ContinueOnError().
		NamespaceParam(cmdNamespace).DefaultNamespace().
		FilenameParam(enforceNamespace, options.Filenames...).
		ResourceTypeOrNameArgs(false, args...).
		Flatten().
		Do()
	err = r.Err()
	if err != nil {
		return err
	}

	infos, err := r.Infos()
	if err != nil {
		return err
	}
	if len(infos) > 1 {
		return fmt.Errorf("multiple resources provided")
	}
	info := infos[0]
	name, namespace := info.Name, info.Namespace
	mapping := info.ResourceMapping()
	client, err := f.RESTClient(mapping)
	if err != nil {
		return err
	}

	helper := resource.NewHelper(client, mapping)
	_, err = helper.Patch(namespace, name, api.StrategicMergePatchType, patchBytes)
	if err != nil {
		return err
	}
	cmdutil.PrintSuccess(mapper, shortOutput, out, "", name, "patched")
	return nil
}
