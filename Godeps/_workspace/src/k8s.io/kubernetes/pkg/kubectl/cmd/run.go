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
	"os"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/api"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/kubectl"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/kubectl/resource"
	"k8s.io/kubernetes/pkg/labels"
)

const (
	run_long = `Create and run a particular image, possibly replicated.
Creates a replication controller to manage the created container(s).`
	run_example = `# Start a single instance of nginx.
$ kubectl run nginx --image=nginx

# Start a single instance of hazelcast and let the container expose port 5701 .
$ kubectl run hazelcast --image=hazelcast --port=5701

# Start a single instance of hazelcast and set environment variables "DNS_DOMAIN=cluster" and "POD_NAMESPACE=default" in the container.
$ kubectl run hazelcast --image=hazelcast --env="DNS_DOMAIN=cluster" --env="POD_NAMESPACE=default"

# Start a replicated instance of nginx.
$ kubectl run nginx --image=nginx --replicas=5

# Dry run. Print the corresponding API objects without creating them.
$ kubectl run nginx --image=nginx --dry-run

# Start a single instance of nginx, but overload the spec of the replication controller with a partial set of values parsed from JSON.
$ kubectl run nginx --image=nginx --overrides='{ "apiVersion": "v1", "spec": { ... } }'

# Start a single instance of nginx and keep it in the foreground, don't restart it if it exits.
$ kubectl run -i -tty nginx --image=nginx --restart=Never

# Start the nginx container using the default command, but use custom arguments (arg1 .. argN) for that command.
$ kubectl run nginx --image=nginx -- <arg1> <arg2> ... <argN>

# Start the nginx container using a different command and custom arguments
$ kubectl run nginx --image=nginx --command -- <cmd> <arg1> ... <argN>`
)

func NewCmdRun(f *cmdutil.Factory, cmdIn io.Reader, cmdOut, cmdErr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use: "run NAME --image=image [--env=\"key=value\"] [--port=port] [--replicas=replicas] [--dry-run=bool] [--overrides=inline-json]",
		// run-container is deprecated
		Aliases: []string{"run-container"},
		Short:   "Run a particular image on the cluster.",
		Long:    run_long,
		Example: run_example,
		Run: func(cmd *cobra.Command, args []string) {
			err := Run(f, cmdIn, cmdOut, cmdErr, cmd, args)
			cmdutil.CheckErr(err)
		},
	}
	cmdutil.AddPrinterFlags(cmd)
	cmd.Flags().String("generator", "", "The name of the API generator to use.  Default is 'run/v1' if --restart=Always, otherwise the default is 'run-pod/v1'.")
	cmd.Flags().String("image", "", "The image for the container to run.")
	cmd.MarkFlagRequired("image")
	cmd.Flags().IntP("replicas", "r", 1, "Number of replicas to create for this container. Default is 1.")
	cmd.Flags().Bool("dry-run", false, "If true, only print the object that would be sent, without sending it.")
	cmd.Flags().String("overrides", "", "An inline JSON override for the generated object. If this is non-empty, it is used to override the generated object. Requires that the object supply a valid apiVersion field.")
	cmd.Flags().StringSlice("env", []string{}, "Environment variables to set in the container")
	cmd.Flags().Int("port", -1, "The port that this container exposes.")
	cmd.Flags().Int("hostport", -1, "The host port mapping for the container port. To demonstrate a single-machine container.")
	cmd.Flags().StringP("labels", "l", "", "Labels to apply to the pod(s).")
	cmd.Flags().BoolP("stdin", "i", false, "Keep stdin open on the container(s) in the pod, even if nothing is attached.")
	cmd.Flags().Bool("tty", false, "Allocated a TTY for each container in the pod.  Because -t is currently shorthand for --template, -t is not supported for --tty. This shorthand is deprecated and we expect to adopt -t for --tty soon.")
	cmd.Flags().Bool("attach", false, "If true, wait for the Pod to start running, and then attach to the Pod as if 'kubectl attach ...' were called.  Default false, unless '-i/--interactive' is set, in which case the default is true.")
	cmd.Flags().String("restart", "Always", "The restart policy for this Pod.  Legal values [Always, OnFailure, Never].  If set to 'Always' a replication controller is created for this pod, if set to OnFailure or Never, only the Pod is created and --replicas must be 1.  Default 'Always'")
	cmd.Flags().Bool("command", false, "If true and extra arguments are present, use them as the 'command' field in the container, rather than the 'args' field which is the default.")
	cmd.Flags().String("requests", "", "The resource requirement requests for this container.  For example, 'cpu=100m,memory=256Mi'")
	cmd.Flags().String("limits", "", "The resource requirement limits for this container.  For example, 'cpu=200m,memory=512Mi'")
	return cmd
}

func Run(f *cmdutil.Factory, cmdIn io.Reader, cmdOut, cmdErr io.Writer, cmd *cobra.Command, args []string) error {
	if len(os.Args) > 1 && os.Args[1] == "run-container" {
		printDeprecationWarning("run", "run-container")
	}

	if len(args) == 0 {
		return cmdutil.UsageError(cmd, "NAME is required for run")
	}

	interactive := cmdutil.GetFlagBool(cmd, "stdin")
	tty := cmdutil.GetFlagBool(cmd, "tty")
	if tty && !interactive {
		return cmdutil.UsageError(cmd, "-i/--stdin is required for containers with --tty=true")
	}
	replicas := cmdutil.GetFlagInt(cmd, "replicas")
	if interactive && replicas != 1 {
		return cmdutil.UsageError(cmd, fmt.Sprintf("-i/--stdin requires that replicas is 1, found %d", replicas))
	}

	namespace, _, err := f.DefaultNamespace()
	if err != nil {
		return err
	}

	restartPolicy, err := getRestartPolicy(cmd, interactive)
	if err != nil {
		return err
	}
	if restartPolicy != api.RestartPolicyAlways && replicas != 1 {
		return cmdutil.UsageError(cmd, fmt.Sprintf("--restart=%s requires that --repliacs=1, found %d", restartPolicy, replicas))
	}
	generatorName := cmdutil.GetFlagString(cmd, "generator")
	if len(generatorName) == 0 {
		if restartPolicy == api.RestartPolicyAlways {
			generatorName = "run/v1"
		} else {
			generatorName = "run-pod/v1"
		}
	}
	generator, found := f.Generator(generatorName)
	if !found {
		return cmdutil.UsageError(cmd, fmt.Sprintf("Generator: %s not found.", generatorName))
	}
	names := generator.ParamNames()
	params := kubectl.MakeParams(cmd, names)
	params["name"] = args[0]
	if len(args) > 1 {
		params["args"] = args[1:]
	}

	params["env"] = cmdutil.GetFlagStringSlice(cmd, "env")

	err = kubectl.ValidateParams(names, params)
	if err != nil {
		return err
	}

	obj, err := generator.Generate(params)
	if err != nil {
		return err
	}

	mapper, typer := f.Object()
	version, kind, err := typer.ObjectVersionAndKind(obj)
	if err != nil {
		return err
	}

	inline := cmdutil.GetFlagString(cmd, "overrides")
	if len(inline) > 0 {
		obj, err = cmdutil.Merge(obj, inline, kind)
		if err != nil {
			return err
		}
	}

	mapping, err := mapper.RESTMapping(kind, version)
	if err != nil {
		return err
	}
	client, err := f.RESTClient(mapping)
	if err != nil {
		return err
	}

	// TODO: extract this flag to a central location, when such a location exists.
	if !cmdutil.GetFlagBool(cmd, "dry-run") {
		data, err := mapping.Codec.Encode(obj)
		if err != nil {
			return err
		}
		obj, err = resource.NewHelper(client, mapping).Create(namespace, false, data)
		if err != nil {
			return err
		}
	}

	attachFlag := cmd.Flags().Lookup("attach")
	attach := cmdutil.GetFlagBool(cmd, "attach")

	if !attachFlag.Changed && interactive {
		attach = true
	}

	if attach {
		opts := &AttachOptions{
			In:    cmdIn,
			Out:   cmdOut,
			Err:   cmdErr,
			Stdin: interactive,
			TTY:   tty,

			Attach: &DefaultRemoteAttach{},
		}
		config, err := f.ClientConfig()
		if err != nil {
			return err
		}
		opts.Config = config

		client, err := f.Client()
		if err != nil {
			return err
		}
		opts.Client = client
		// TODO: this should be abstracted into Factory to support other types
		switch t := obj.(type) {
		case *api.ReplicationController:
			return handleAttachReplicationController(client, t, opts)
		case *api.Pod:
			return handleAttachPod(client, t, opts)
		default:
			return fmt.Errorf("cannot attach to %s: not implemented", kind)
		}
	}

	outputFormat := cmdutil.GetFlagString(cmd, "output")
	if outputFormat != "" {
		return f.PrintObject(cmd, obj, cmdOut)
	}
	cmdutil.PrintSuccess(mapper, false, cmdOut, mapping.Resource, args[0], "created")
	return nil
}

func waitForPodRunning(c *client.Client, pod *api.Pod, out io.Writer) (status api.PodPhase, err error) {
	for {
		pod, err := c.Pods(pod.Namespace).Get(pod.Name)
		if err != nil {
			return api.PodUnknown, err
		}
		ready := false
		if pod.Status.Phase == api.PodRunning {
			ready = true
			for _, status := range pod.Status.ContainerStatuses {
				if !status.Ready {
					ready = false
					break
				}
			}
			if ready {
				return api.PodRunning, nil
			}
		}
		if pod.Status.Phase == api.PodSucceeded || pod.Status.Phase == api.PodFailed {
			return pod.Status.Phase, nil
		}
		fmt.Fprintf(out, "Waiting for pod %s/%s to be running, status is %s, pod ready: %v\n", pod.Namespace, pod.Name, pod.Status.Phase, ready)
		time.Sleep(2 * time.Second)
		continue
	}
}

func handleAttachReplicationController(c *client.Client, controller *api.ReplicationController, opts *AttachOptions) error {
	var pods *api.PodList
	for pods == nil || len(pods.Items) == 0 {
		var err error
		if pods, err = c.Pods(controller.Namespace).List(labels.SelectorFromSet(controller.Spec.Selector), fields.Everything()); err != nil {
			return err
		}
		if len(pods.Items) == 0 {
			fmt.Fprint(opts.Out, "Waiting for pod to be scheduled\n")
			time.Sleep(2 * time.Second)
		}
	}
	pod := &pods.Items[0]
	return handleAttachPod(c, pod, opts)
}

func handleAttachPod(c *client.Client, pod *api.Pod, opts *AttachOptions) error {
	status, err := waitForPodRunning(c, pod, opts.Out)
	if err != nil {
		return err
	}
	if status == api.PodSucceeded || status == api.PodFailed {
		return handleLog(c, pod.Namespace, pod.Name, &api.PodLogOptions{Container: opts.GetContainerName(pod)}, opts.Out)
	}
	opts.Client = c
	opts.PodName = pod.Name
	opts.Namespace = pod.Namespace
	if err := opts.Run(); err != nil {
		fmt.Fprintf(opts.Out, "Error attaching, falling back to logs: %v\n", err)
		return handleLog(c, pod.Namespace, pod.Name, &api.PodLogOptions{Container: opts.GetContainerName(pod)}, opts.Out)
	}
	return nil
}

func getRestartPolicy(cmd *cobra.Command, interactive bool) (api.RestartPolicy, error) {
	restart := cmdutil.GetFlagString(cmd, "restart")
	if len(restart) == 0 {
		if interactive {
			return api.RestartPolicyOnFailure, nil
		} else {
			return api.RestartPolicyAlways, nil
		}
	}
	switch api.RestartPolicy(restart) {
	case api.RestartPolicyAlways:
		return api.RestartPolicyAlways, nil
	case api.RestartPolicyOnFailure:
		return api.RestartPolicyOnFailure, nil
	case api.RestartPolicyNever:
		return api.RestartPolicyNever, nil
	default:
		return "", cmdutil.UsageError(cmd, fmt.Sprintf("invalid restart policy: %s", restart))
	}
}
