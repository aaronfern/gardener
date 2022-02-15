# Local Provider Extension

The "local provider" extension is used to allow the usage of seed and shoot clusters which run entirely locally without any real infrastructure or cloud provider involved.
It implements Gardener's extension contract ([GEP-1](../proposals/01-extensibility.md)) and thus comprises several controllers and webhooks acting on resources in seed and shoot clusters.

The code is maintained in [`pkg/provider-local`](../../pkg/provider-local).

## Motivation

The motivation for maintaining such extension is the following:

- 🛡 Output Qualification: Run fast and cost-efficient end-to-end tests, locally and in CI systems (increased confidence ⛑ before merging pull requests)
- ⚙️ Development Experience: Develop Gardener entirely on a local machine without any external resources involved (improved costs 💰 and productivity 🚀)
- 🤝 Open Source: Quick and easy setup for a first evaluation of Gardener and a good basis for first contributions

## Current Limitations

The following enlists the current limitations of the implementation.
Please note that all of them are no technical limitations/blockers but simply advanced scenarios that we haven't had invested yet into.

1. Shoot clusters can only have one node when `.spec.networking.type=local`.

   _We use [kindnetd](https://github.com/kubernetes-sigs/kind/blob/main/images/kindnetd/README.md) as CNI plugin in shoot clusters and didn't invest into making it work with multiple worker nodes._

2. `NetworkPolicy`s are not effective.

   _`kindnetd` does not ship any controller for Kubernetes `NetworkPolicy`s, hence, they are not effective. Typically, the same applies for the local seed cluster unless a different CNI plugin is pro-actively installed._

3. Shoot clusters don't support persistent storage.

   _We don't install any CSI plugin into the shoot cluster yet, hence, there is no persistent storage for shoot clusters._

4. No support for ETCD backups.

   _We have not yet implemented the [`BackupBucket`](backupbucket.md)/[`BackupEntry`](backupentry.md) extension API, hence, there is no support for ETCD backups._

5. No owner TXT `DNSRecord`s (hence, no ["bad-case" control plane migration](../proposals/17-shoot-control-plane-migration-bad-case.md)).

   _In order to realize DNS (see [Implementation Details](#implementation-details) section below), the `/etc/hosts` file is manipulated. This does not work for TXT records. In the future, we could look into using [CoreDNS](https://coredns.io/) instead._

6. No load balancers for Shoot clusters.

   _We have not yet developed a `cloud-controller-manager` which could reconcile load balancer `Service`s in the shoot cluster. Hence, when the gardenlet's `ReversedVPN` feature gate is disabled then the `kube-system/vpn-shoot` `Service` must be manually patched (with `{"status": {"loadBalancer": {"ingress": [{"hostname": "vpn-shoot"}]}}}`) to make the reconciliation work._

7. Only one shoot cluster possible when gardenlet's `APIServerSNI` feature gate is disabled.

   _When [`APIServerSNI`](../proposals/08-shoot-apiserver-via-sni.md) is disabled then gardenlet uses load balancer `Service`s in order to expose the shoot clusters' `kube-apiserver`s. Typically, local Kubernetes clusters don't support this. In this case, the local extension uses the host IP to expose the `kube-apiserver`, however, this can only be done once._

8. Dependency-Watchdog cannot be enabled.

   _The `dependency-watchdog` needs to be able to resolve the shoot cluster's DNS names. It is not yet able to do so, hence, it cannot be enabled._

9. `Ingress`es exposed in the seed cluster are not reachable.

   _There is no DNS resolution for the domains used for `Ingress`es in the seed cluster yet, hence, they are not reachable. Consequently, the [shoot node logging](../deployment/configuring_logging.md#enable-logs-from-the-shoots-node-systemd-services) feature does not work end-to-end._

## Implementation Details

This section contains information about how the respective controllers and webhooks are implemented and what their purpose is.

### Bootstrapping

The Helm chart of the `provider-local` extension defined in its [`ControllerDeployment`](controllerregistration.md) contains a special deployment for a [CoreDNS](https://coredns.io/) instance in a `gardener-extension-provider-local-coredns` namespace in the seed cluster.

This CoreDNS instance is responsible for enabling the components running in the shoot clusters to be able to resolve the DNS names when they communicate with their `kube-apiserver`s.

It contains static configuration to resolve the DNS names based on `local.gardener.cloud` to either the `istio-ingressgateway.istio-ingress.svc` or the `kube-apiserver.<shoot-namespace>.svc` addresses (depending on whether the `--apiserver-sni-enabled` flag is set to `true` or `false`).

### Controllers

There are controllers for all resources in the `extensions.gardener.cloud/v1alpha1` API group except for `BackupBucket` and `BackupEntry`s.

#### `ControlPlane`

This controller is not deploying anything since we haven't invested yet into a `cloud-controller-manager` or CSI solution.
For the latter, we could probably use the [local-path-provisioner](https://github.com/rancher/local-path-provisioner).

#### `DNSRecord`

This controller manipulates the `/etc/hosts` file and adds a new line for each `DNSRecord` it observes.
This enables accessing the shoot clusters from the respective machine, however, it also requires to run the extension with elevated privileges (`sudo`).

The `/etc/hosts` would be extended as follows:

```text
# Begin of gardener-extension-provider-local section
10.84.23.24 api.local.local.external.local.gardener.cloud
10.84.23.24 api.local.local.internal.local.gardener.cloud
...
# End of gardener-extension-provider-local section
```

#### `Infrastructure`

This controller generates a `NetworkPolicy` which allows the control plane pods (like `kube-apiserver`) to communicate with the worker machine pods (see [`Worker` section](#worker))).

In addition, it creates a `Service` named `vpn-shoot` which is only used in case the gardenlet's `ReversedVPN` feature gate is disabled.
This `Service` enables the `vpn-seed` containers in the `kube-apiserver` pods in the seed cluster to communicate with the `vpn-shoot` pod running in the shoot cluster.

#### `Network`

This controller deploys a `ManagedResource` which contains the [kindnetd](https://github.com/kubernetes-sigs/kind/blob/main/images/kindnetd/README.md) `DaemonSet` which is used as CNI in the shoot clusters.

#### `OperatingSystemConfig`

This controller leverages the standard [`oscommon` library](../../extensions/pkg/controller/operatingsystemconfig/oscommon) in order to render a simple cloud-init template which can later be executed by the shoot worker nodes.

The shoot worker nodes are `Pod`s with a container based on the `kindest/node` image. This is maintained in https://github.com/gardener/machine-controller-manager-provider-local/tree/master/node and has a special `run-userdata` systemd service which executes the cloud-init generated earlier by the `OperatingSystemConfig` controller.

#### `Worker`

This controller leverages the standard [generic `Worker` actuator](../../extensions/pkg/controller/worker/genericactuator) in order to deploy the [`machine-controller-manager`](https://github.com/gardener/machine-controller-manager) as well as the [`machine-controller-manager-provider-local`](https://github.com/gardener/machine-controller-manager-provider-local).

Additionally, it generates the [`MachineClass`es](https://github.com/gardener/machine-controller-manager-provider-local/blob/master/kubernetes/machine-class.yaml) and the `MachineDeployment`s based on the specification of the `Worker` resources.

#### `DNSProvider`

Due to legacy reasons, the gardenlet still creates `DNSProvider` resources part of the [`dns.gardener.cloud/v1alpha1` API group](https://github.com/gardener/external-dns-management/).
Since those are only needed in conjunction with the [`shoot-dns-service` extension](https://github.com/gardener/gardener-extension-shoot-dns-service) and have no relevance for the local provider, it just sets their `status.state=Ready` to please the expectations.
In the future, this controller can be dropped when the gardenlet no longer creates such `DNSProvider`s.

#### `Service`

This controller reconciles the `istio-ingress/istio-ingressgateway` `Service` in the seed cluster if the `--apiserver-sni-enabled` flag is set to `true` (default).
Otherwise, it reconciles the `kube-apiserver` `Service` in the shoot namespaces in the seed cluster.

All such `Service`s are of type `LoadBalancer`.
Since the local Kubernetes clusters used as seed typically don't support such services, this controller sets the `.status.ingress.loadBalancer.ip[0]` to the IP of the host.

#### `Node`

This controller reconciles the `Node`s of [kind](https://kind.sigs.k8s.io/) clusters.
Typically, the `.status.{capacity,allocatable}` values are determined by the resources configured for the Docker daemon (see for example [this](https://docs.docker.com/desktop/mac/#resources) for Mac).
Since many of the `Pod`s deployed by Gardener have quite high `.spec.resources.{requests,limits}`, the kind `Node`s easily get filled up and only a few `Pod`s can be scheduled (even if they barely consume any of their reserved resources).
In order to improve the user experience, the controller submits an empty patch which triggers the "Node webhook" (see below section) in case the `.status.{capacity,allocatable}` values are not high enough.
The webhook will increase the capacity of the `Node`s to allow all `Pod`s to be scheduled.

#### Health Checks

The health check controller leverages the [health check library](healthcheck-library.md) in order to

- check the health of the `ManagedResource/extension-controlplane-shoot-webhooks` and populate the `SystemComponentsHealthy` condition in the `ControlPlane` resource.
- check the health of the `ManagedResource/extension-networking-local` and populate the `SystemComponentsHealthy` condition in the `Network` resource.
- check the health of the `ManagedResource/extension-worker-mcm-shoot` and populate the `SystemComponentsHealthy` condition in the `Worker` resource.
- check the health of the `Deployment/machine-controller-manager` and populate the `ControlPlaneHealthy` condition in the `Worker` resource.
- check the health of the `Node`s and populate the `EveryNodeReady` condition in the `Worker` resource.

### Webhooks

#### Control Plane

This webhook reacts on the `OperatingSystemConfig` containing the configuration of the kubelet and sets the `failSwapOn` to `false` (independent of what is configured in the `Shoot` spec) ([ref](https://github.com/kubernetes-sigs/kind/blob/b6bc112522651d98c81823df56b7afa511459a3b/site/content/docs/design/node-image.md#design)).

#### Control Plane Exposure

This webhook reacts on the `kube-apiserver` `Service` in shoot namespaces in the seed in case the gardenlet's `APIServerSNI` feature gate is disabled.
It sets the `nodePort` to `30443` to enable communication from the host (this requires a port mapping to work when creating the local cluster).

#### Machine Pod

This webhook reacts on `Pod`s created when the `machine-controller-manager` reconciles `Machine`s.
It sets the `.spec.dnsPolicy=None` and `.spec.dnsConfig.nameServers` to the cluster IP of the `coredns` `Service` created in the `gardener-extension-provider-local-coredns` namespaces (see the [Bootstrapping section](#bootstrapping) for more details).

#### Node

This webhook reacts on [kind](https://kind.sigs.k8s.io/) `Node`s and sets the `.status.{allocatable,capacity}.cpu="100"` and `.status.{allocatable,capacity}.memory="100Gi"` fields.
See also the above section about the "Node controller" for more information.

#### Shoot

This webhook reacts on the `ConfigMap` used by the `kube-proxy` and sets the `maxPerCore` field to `0` since other values don't work well in conjunction with the `kindest/node` image which is used as base for the shoot worker machine pods ([ref](https://github.com/kubernetes-sigs/kind/blob/fa7d86470f4c0e924fc4c2e767ec8491c45f4304/pkg/cluster/internal/kubeadm/config.go#L283-L285)).

## Future Work

Future work could mostly focus on resolving above listed [limitations](#limitations), i.e.,

- Add storage support for shoot clusters.
- Implement a `cloud-controller-manager` and deploy it via the [`ControlPlane` controller](#controlplane).
- Implement support for `BackupBucket` and `BackupEntry`s to enable ETCD backups for shoot clusters (based on the support for local disks in [`etcd-backup-restore`](https://github.com/gardener/etcd-backup-restore)).
- Switch from `kindnetd` to a different CNI plugin which supports `NetworkPolicy`s.
- Properly implement `.spec.machineTypes` in the `CloudProfile`s (i.e., configure `.spec.resources` properly for the created shoot worker machine pods).