# kubectl-enter
kubectl-enter is a kubectl plugin to gain ssh like access to a specific node in your k8s cluster. This plugin does not require any direct access to a node, you only need to be able talk to k8s api and be able to spawn pod on the privileged pod with hostPID security setting. You will gain root access on the under root user.

Plugin will use default kubectl context and kubeconfig located in `$HOME/.kube/config`

## requirements
- `kubectl` - needs to be in a path as the plugin is calling `kubectl` without any absolute path

## how to install
- clone the repo and build the plugin with `go build`
- copy plugin into path `cp ./kubectl-enter /usr/local/bin/`

##  how to run
```
kubectl enter my-node-name
```
node name can get obtained from the `kubectl get node` command

## configuration

- DockerRegistry - `KUBECTL_ENTER_REGISTRY` - specify docker registry for the pod image that is used for spawning the pod, defaults to 'docker.io', should not need changes unless the registry is blocked
- ServiceAccount - `KUBECTL_ENTER_SA` - specify which service account should be used to be able spawn privileged pod with hostPID settings, this is necessery in case your cluster is running strict PSP. IF you do not enforce PSP you could set to 'default'.The default settings is set to 'kube-proxy' as this service account has often enought privileges for this purpose.


## implementation details
the access is gained by running a privileged pod with hostPID security settings and than running `kubectl exec` command towards that pod and executing `nsenter` cmmand with specific settings to run `/bin/bash` in the OS root namespace (thus geeting out of the container bounds)


## custom service account
If your cluster uses strict PSP policies and you do not have any precreated PSP with binding to an service account here is  guide how to do it:

apply this to your cluster and set env variable `KUBECTL_ENTER_SA` to `kubectl-enter`

```
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  annotations:
  name: kubectl-enter
spec:
  allowPrivilegeEscalation: true
  allowedCapabilities:
  - '*'
  fsGroup:
    rule: RunAsAny
  hostIPC: true
  hostPID: true
  privileged: true
  runAsUser:
    rule: RunAsAny
  seLinux:
    rule: RunAsAny
  supplementalGroups:
    rule: RunAsAny
  volumes:
  - '*'
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kubectl-enter
rules:
- apiGroups:
  - extensions
  resourceNames:
  - kubectl-enter
  resources:
  - podsecuritypolicies
  verbs:
  - use
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kubectl-enter
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: kubectl-enter
subjects:
- kind: ServiceAccount
  name: kubectl-enter
  namespace: kube-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kubectl-enter
  namespace: kube-system

```
