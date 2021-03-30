package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/giantswarm/microerror"
	flag "github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

type Flag struct {
	Namespace      string
	DockerRegistry string
	ServiceAccount string
}

func main() {
	err := mainError()
	if err != nil {
		panic(fmt.Sprintf("%#v", err))
	}
}

func mainError() error {
	var err error

	var f Flag
	flag.StringVar(&f.DockerRegistry, "registry", os.Getenv("KUBECTL_ENTER_REGISTRY"), "registry for the sleeper container, also can be set via env KUBECTL_ENTER_REGISTRY, defaults to 'docker.io'")
	flag.StringVar(&f.ServiceAccount, "service-account", os.Getenv("KUBECTL_ENTER_SA"), "service account that has privileges to run privileged container with host pid, also can be set via env KUBECTL_ENTER_SA, defaults to 'kube-proxy'")

	if f.DockerRegistry == "" {
		f.DockerRegistry = "docker.io"
	}
	if f.ServiceAccount == "" {
		f.ServiceAccount = "kube-proxy"
	}

	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("exec-to-node: - 0.0.1")
		return nil
	}

	if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		fmt.Print("Kubectl 'enter' plugin will give ssh like access to a node\n")
		fmt.Print("How to run: ./kubectl enter my-node-name\n")

		flag.Usage()
		return nil
	}

	flag.Parse()

	nodeName := os.Args[1]

	ctx := context.Background()

	client, err := GetCtrlClient()
	if err != nil {
		return microerror.Mask(err)
	}

	node := &corev1.Node{}
	err = client.Get(ctx, ctrl.ObjectKey{Name: nodeName}, node)
	if err != nil {
		return microerror.Mask(err)
	}

	pod := podSpec(nodeName, f.DockerRegistry, f.ServiceAccount)

	err = client.Create(ctx, pod)
	if errors.IsAlreadyExists(err) {
		// pod already exists, lets fall thru and use it
	} else if err != nil {
		return microerror.Mask(err)
	}
	fmt.Printf("Creating helper pod on node and waiting for running condition.\n")
	for {
		err = client.Get(ctx, ctrl.ObjectKey{Name: pod.Name, Namespace: pod.Namespace}, pod)
		if pod.Status.Phase == "Running" {
			break
		}
		if err != nil {
			fmt.Printf("error %s", err)
		}
	}
	fmt.Printf("Helper pod is running on the node.\n")

	err = runExecCommand(pod.Name, pod.Namespace)
	if err != nil {
		return microerror.Mask(err)
	}

	err = client.Delete(ctx, pod)
	if err != nil {
		return microerror.Mask(err)
	}
	fmt.Printf("Deleting helper pod.\n")

	for {
		err = client.Get(ctx, ctrl.ObjectKey{Name: pod.Name, Namespace: pod.Namespace}, pod)
		if errors.IsNotFound(err) {
			break
		}
	}
	fmt.Printf("Deleted helper pod.\n")

	return nil
}

func runExecCommand(podName, ns string) error {
	args := []string{
		"-n",
		ns,
		"exec",
		"-it",
		podName,
		"--",
		"nsenter",
		"-t",
		"1",
		"-m",
		"-u",
		"-n",
		"-i",
		"--",
		"/bin/bash",
	}

	cmd := exec.Command("kubectl", args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()

	if err != nil {
		fmt.Printf("%s\n", err)
	}
	return nil
}
