package main

import (
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/scheme"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func GetCtrlClient() (ctrl.Client, error) {
	schemes := []func(*runtime.Scheme) error{
		corev1.AddToScheme,
	}

	// Extend the global client-go scheme which is used by all the tools under
	// the hood. The scheme is required for the controller-runtime controller to
	// be able to watch for runtime objects of a certain type.
	schemeBuilder := runtime.SchemeBuilder(schemes)

	err := schemeBuilder.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	restConfig, err := config.GetConfig()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Configure a dynamic rest mapper to the controller client so it can work
	// with runtime objects of arbitrary types. Note that this is the default
	// for controller clients created by controller-runtime managers.
	// Anticipating a rather uncertain future and more breaking changes to come
	// we want to separate client and manager. Thus we configure the client here
	// properly on our own instead of relying on the manager to provide a
	// client, which might change in the future.
	mapper, err := apiutil.NewDynamicRESTMapper(rest.CopyConfig(restConfig))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	client, err := ctrl.New(restConfig, ctrl.Options{Scheme: scheme.Scheme, Mapper: mapper})
	if err != nil {
		return nil, microerror.Mask(err)
	}
	return client, nil
}
