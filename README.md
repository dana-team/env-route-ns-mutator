# env-route-ns-mutator

This project implements a Kubernetes admission webhook that mutates `Namespace` objects and `Route` objects in OpenShift. It does so based on the environment the `Namespace` or the `Route` is a part of.

The list of respected environments is set by the `environments` env var set on the `manager` deployment.

## Namespace Mutator

The mutator adds an `environment: <ENV>` label to every Namespace that has the `defaultTolerations` annotation that matches the specific environment:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: test-ns
  labels: {} # original
  annotations:
    scheduler.alpha.kubernetes.io/defaultTolerations: "[{"operator": "Exists", "effect": "NoSchedule", "key": "<ENV>"}]"
```

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: test-ns
  labels:
    environment: <ENV> # mutated
  annotations:
    scheduler.alpha.kubernetes.io/defaultTolerations: "[{"operator": "Exists", "effect": "NoSchedule", "key": "<ENV>"}]"
```

## Route Mutator

The mutator changes the `host` field of the `Route` based on the `environment: <ENV>` label on the `namespace` the `Route` exists in. 

For example, it would change the `apps` part of the `Route` to be `<ENV>-apps`.

### Empty Host

```yaml
kind: Route
apiVersion: route.openshift.io/v1
metadata:
  name: route-test
  namespace: test-ns
spec:
  host: "" # (original)
```

```yaml
kind: Route
apiVersion: route.openshift.io/v1
metadata:
  name: route-test
  namespace: test-ns
spec:
  host: "route-test-test-ns.<ENV>-apps.cluster-name.example.dom" # (mutated)
```

### Host with Cluster Ingress Domain

```yaml
kind: Route
apiVersion: route.openshift.io/v1
metadata:
  name: route-test
  namespace: test-ns
spec:
  host: "test.apps.cluster-name.example.dom" # (original)
```

```yaml
kind: Route
apiVersion: route.openshift.io/v1
metadata:
  name: route-test
  namespace: test-ns
spec:
  host: "test.<ENV>-apps.cluster-name.example.dom" # (mutated)
```

## Getting started

### Deploying the controller

```bash
$ make deploy IMG=ghcr.io/dana-team/env-route-ns-mutator:<release>
```

### Install with Helm

Helm chart docs are available on `charts/env-route-ns-mutator` directory.

Make sure `cert-manager` is [installed](https://cert-manager.io/docs/installation/helm/) as a prerequisite.

```
$ helm upgrade --install env-route-ns-mutator --namespace env-route-ns-mutator-system --create-namespace oci://ghcr.io/dana-team/helm-charts/env-route-ns-mutator --version <release>
```

#### Build your own image

```bash
$ make docker-build docker-push IMG=<registry>/env-route-ns-mutator:<tag>
```