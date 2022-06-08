package main

import (
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/kubectl/pkg/scheme"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
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

	client, err := ctrl.New(restConfig, ctrl.Options{})
	if err != nil {
		return nil, microerror.Mask(err)
	}
	return client, nil
}
