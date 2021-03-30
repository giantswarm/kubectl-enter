package main

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	runCommandDockerImage   = "giantswarm/alpine:3.11.6"
	runCommandNamespace     = metav1.NamespaceSystem
	runCommandPriorityClass = "system-cluster-critical"
	runCommandSAName        = "kube-proxy"
)

func podSpec(nodeName string, dockerRegistry string) *corev1.Pod {
	privileged := true
	priority := int32(2000000000)
	podName := fmt.Sprintf("exec-to-node-%s-helper", nodeName)
	cpu := resource.MustParse("50m")
	memory := resource.MustParse("50Mi")

	p := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: corev1.GroupName,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: runCommandNamespace,
			Labels: map[string]string{
				"created-by": "kubectl-plugin-exec-to-node",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "sleeper",
					Image: jobDockerImage(dockerRegistry),
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    cpu,
							corev1.ResourceMemory: memory,
						},
					},
					SecurityContext: &corev1.SecurityContext{
						Privileged:               &privileged,
						AllowPrivilegeEscalation: &privileged,
					},
					Command: []string{
						"/bin/sh",
						"-c",
						"sleep 100000",
					},
				},
			},
			HostPID: true,
			NodeSelector: map[string]string{
				"kubernetes.io/hostname": nodeName,
			},
			RestartPolicy:      corev1.RestartPolicyNever,
			Priority:           &priority,
			PriorityClassName:  runCommandPriorityClass,
			ServiceAccountName: runCommandSAName,
			Tolerations: []corev1.Toleration{
				{
					Key:      "node.kubernetes.io/unschedulable",
					Operator: "Exists",
					Effect:   "NoSchedule",
				},
				{
					Key:    "node-role.kubernetes.io/master",
					Effect: "NoSchedule",
				},
				{
					Effect:   "NoSchedule",
					Operator: "Exists",
				},
			},
		},
	}
	return &p
}

func jobDockerImage(registry string) string {
	return fmt.Sprintf("%s/%s", registry, runCommandDockerImage)
}
