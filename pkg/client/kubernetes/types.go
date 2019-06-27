// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kubernetes

import (
	"context"

	dnsscheme "github.com/gardener/external-dns-management/pkg/client/dns/clientset/versioned/scheme"
	gardencoreclientset "github.com/gardener/gardener/pkg/client/core/clientset/versioned"
	gardencorescheme "github.com/gardener/gardener/pkg/client/core/clientset/versioned/scheme"
	gardenextensionsscheme "github.com/gardener/gardener/pkg/client/extensions/clientset/versioned/scheme"
	gardenclientset "github.com/gardener/gardener/pkg/client/garden/clientset/versioned"
	gardenscheme "github.com/gardener/gardener/pkg/client/garden/clientset/versioned/scheme"
	machineclientset "github.com/gardener/gardener/pkg/client/machine/clientset/versioned"
	machinescheme "github.com/gardener/gardener/pkg/client/machine/clientset/versioned/scheme"

	resourcesscheme "github.com/gardener/gardener-resource-manager/pkg/apis/resources/v1alpha1"

	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiextensionsscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	kubernetesclientset "k8s.io/client-go/kubernetes"
	corescheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	apiregistrationclientset "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	apiregistrationscheme "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// CronJobs is a constant for a Kubernetes resource with the same name.
	CronJobs = "cronjobs"

	// CustomResourceDefinitions is a constant for a Kubernetes resource with the same name.
	CustomResourceDefinitions = "customresourcedefinitions"

	// DaemonSets is a constant for a Kubernetes resource with the same name.
	DaemonSets = "daemonsets"

	// Deployments is a constant for a Kubernetes resource with the same name.
	Deployments = "deployments"

	// Ingresses is a constant for a Kubernetes resource with the same name.
	Ingresses = "ingresses"

	// Jobs is a constant for a Kubernetes resource with the same name.
	Jobs = "jobs"

	// Namespaces is a constant for a Kubernetes resource with the same name.
	Namespaces = "namespaces"

	// PersistentVolumeClaims is a constant for a Kubernetes resource with the same name.
	PersistentVolumeClaims = "persistentvolumeclaims"

	// PersistentVolumes is a constant for a Kubernetes resource with the same name.
	PersistentVolumes = "persistentvolumes"

	// Pods is a constant for a Kubernetes resource with the same name.
	Pods = "pods"

	// ReplicaSets is a constant for a Kubernetes resource with the same name.
	ReplicaSets = "replicasets"

	// ReplicationControllers is a constant for a Kubernetes resource with the same name.
	ReplicationControllers = "replicationcontrollers"

	// Services is a constant for a Kubernetes resource with the same name.
	Services = "services"

	// StatefulSets is a constant for a Kubernetes resource with the same name.
	StatefulSets = "statefulsets"
)

var (
	// GardenScheme is the scheme used in the Garden cluster.
	GardenScheme = runtime.NewScheme()
	// SeedScheme is the scheme used in the Seed cluster.
	SeedScheme = runtime.NewScheme()
	// ShootScheme is the scheme used in the Shoot cluster.
	ShootScheme = runtime.NewScheme()
	// PlantScheme is the scheme used in the Plant cluster
	PlantScheme = runtime.NewScheme()

	// DefaultDeleteOptionFuncs use foreground propagation policy and grace period of 60 seconds.
	DefaultDeleteOptionFuncs = []client.DeleteOptionFunc{
		client.PropagationPolicy(metav1.DeletePropagationForeground),
		client.GracePeriodSeconds(60),
	}
	// ForceDeleteOptionFuncs use background propagation policy and grace period of 0 seconds.
	ForceDeleteOptionFuncs = []client.DeleteOptionFunc{
		client.PropagationPolicy(metav1.DeletePropagationBackground),
		client.GracePeriodSeconds(0),
	}
)

func init() {
	gardenSchemeBuilder := runtime.NewSchemeBuilder(
		corescheme.AddToScheme,
		gardenscheme.AddToScheme,
		gardencorescheme.AddToScheme,
	)
	utilruntime.Must(gardenSchemeBuilder.AddToScheme(GardenScheme))

	seedSchemeBuilder := runtime.NewSchemeBuilder(
		corescheme.AddToScheme,
		machinescheme.AddToScheme,
		dnsscheme.AddToScheme,
		gardenextensionsscheme.AddToScheme,
		resourcesscheme.AddToScheme,
	)
	utilruntime.Must(seedSchemeBuilder.AddToScheme(SeedScheme))

	shootSchemeBuilder := runtime.NewSchemeBuilder(
		corescheme.AddToScheme,
		apiextensionsscheme.AddToScheme,
		apiregistrationscheme.AddToScheme,
	)
	utilruntime.Must(shootSchemeBuilder.AddToScheme(ShootScheme))

	plantSchemeBuilder := runtime.NewSchemeBuilder(
		corescheme.AddToScheme,
		gardencorescheme.AddToScheme,
	)
	utilruntime.Must(plantSchemeBuilder.AddToScheme(PlantScheme))

}

// Clientset is a struct containing the configuration for the respective Kubernetes
// cluster, the collection of Kubernetes clients <Clientset> containing all REST clients
// for the built-in Kubernetes API groups, and the Garden which is a REST clientset
// for the Garden API group.
// The RESTClient itself is a normal HTTP client for the respective Kubernetes cluster,
// allowing requests to arbitrary URLs.
// The version string contains only the major/minor part in the form <major>.<minor>.
type Clientset struct {
	config     *rest.Config
	restMapper meta.RESTMapper
	restClient rest.Interface

	applier ApplierInterface

	client client.Client

	kubernetes      kubernetesclientset.Interface
	garden          gardenclientset.Interface
	gardenCore      gardencoreclientset.Interface
	machine         machineclientset.Interface
	apiextension    apiextensionsclientset.Interface
	apiregistration apiregistrationclientset.Interface

	version string
}

// Applier is a default implementation of the ApplyInterface. It applies objects with
// by first checking whether they exist and then either creating / updating them (update happens
// with a predefined merge logic).
type Applier struct {
	client    client.Client
	discovery discovery.CachedDiscoveryInterface
}

// MergeFunc determines how oldOj is merged into new oldObj.
type MergeFunc func(newObj, oldObj *unstructured.Unstructured)

// ApplierOptions contains options used by the Applier.
type ApplierOptions struct {
	MergeFuncs map[schema.GroupKind]MergeFunc
}

// ApplierInterface is an interface which describes declarative operations to apply multiple
// Kubernetes objects.
type ApplierInterface interface {
	ApplyManifest(ctx context.Context, unstructured UnstructuredReader, options ApplierOptions) error
	DeleteManifest(ctx context.Context, unstructured UnstructuredReader) error
}

// Interface is used to wrap the interactions with a Kubernetes cluster
// (which are performed with the help of kubernetes/client-go) in order to allow the implementation
// of several Kubernetes versions.
type Interface interface {
	RESTConfig() *rest.Config
	RESTMapper() meta.RESTMapper
	RESTClient() rest.Interface

	Client() client.Client
	Applier() ApplierInterface

	Kubernetes() kubernetesclientset.Interface
	Garden() gardenclientset.Interface
	GardenCore() gardencoreclientset.Interface
	Machine() machineclientset.Interface
	APIExtension() apiextensionsclientset.Interface
	APIRegistration() apiregistrationclientset.Interface

	// Deprecated: Use `Client()` and utils instead.
	ForwardPodPort(string, string, int, int) (chan struct{}, error)
	CheckForwardPodPort(string, string, int, int) error

	Version() string
}
