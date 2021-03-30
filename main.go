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
	flag.StringVar(&f.Namespace, "namespace", "", "ns of the pod")
	flag.StringVar(&f.DockerRegistry, "registry", "docker.io", "registry for the sleepr container")

	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("exec-to-node: - 0.0.1")
		return nil
	}
	if len(os.Args) > 1 && os.Args[1] == "--help" {
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

	pod := podSpec(nodeName, f.DockerRegistry)

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
