# Alerts

In SSC [Prometheus][prometheus] consumes [alerts] via Kubernetes
[custom resource definitions][crd].

> TLDR: See upstream for docs on [design] and [implementation] of the
> [alert crd].

## Prometheus format

[Prometheus][prometheus] describes alerts in a yaml format that looks
like this:

```
groups:
- name: example
  rules:
  - alert: HighErrorRate
    expr: job:request_latency_seconds:mean5m{job="myjob"} > 0.5
    for: 10m
    labels:
      severity: page
    annotations:
      summary: High request latency
```

Here the important parts to note are:

- `alert`: This should be concise and in PascalCase
- `expr`: This is any [promql] expression and should include a
  threshold.

       > Note: The threshold can be the name of another metric or [promql]
       > expression
- `for`: This is a time duration. Once reached, the alert will
  transition from `pending` to `firing`
- `labels`: These are attached to the alert, usually for context or
  routing purposes.

       > Note: Currently routing configuration only **pages** with
       > `severity: critical`
- `annotations`: These are typically used for:
    1. `summary`: Concise summary of the state
    1. `description`: A bit more verbose than `summary`

**Reference:** See [upstream docs][alerts] for more details

## Deployment

Because [alerts] are configured via [crd] in our configuration, you
deploy the alerts via the same mechanism you deploy the service
itself. This means you can deploy using typical tools like:

- kubectl
- ksonnet

The [crd] design means we wrap the upstream format in an envelope for
Kubernetes to consume. Here's the most important part:

```
kind: PrometheusRule
apiVersion: monitoring.coreos.com/v1
metadata:
  name: k8s-demo
  labels:
    role: alert-rules
    prometheus: k8s
```

**Note:**

- `kind` This is a `PrometheusRule` [crd] object
- `name` This is the name of your service. This ultimately gets
  suffixed with `.rules` and written to disk.
- `labels`: Required for registration. In time we will likely have
  more than one prometheus shard running, and you'll specify which one
  to use via the `metadata.prometheus` label.

### Example: kubectl

If you wanted to deploy via kubectl you'd do something like this.
Create a file in your repo named `kubectl/alerts.yaml`:

```
kind: PrometheusRule
apiVersion: monitoring.coreos.com/v1
metadata:
  name: k8s-demo
  labels:
    role: alert-rules
    prometheus: k8s
spec:
  groups:
    - name: k8s-demo.rules
      rules:
        - alert: ElevatedLatency
          expr: |
            histogram_quantile(0.98, sum(increase(k8s_demo_rest_api_histogram_seconds_bucket[5m]))
              by (service_name, le, code, path, method)) > 3
          for: 4m
          labels:
            severity: warning
        - alert: ElevatedLatency
          expr: |
            histogram_quantile(0.98, sum(increase(k8s_demo_rest_api_histogram_seconds_bucket[5m]))
              by (service_name, le, code, path, method)) > 7
          for: 4m
          labels:
            severity: critical
```

Then you'd do a deploy via:

```
kubectl apply -f kubectl/alerts.yaml
```

Within a couple of minutes you'll find your new alerts deployed to
which ever cluster you deployed to. The intention would be to run this
via your CICD pipeline. You can also deploy this locally via Minikube.

### Example: jsonnet

If you wanted to deploy via ksonnet you'd do something like this.
Create a file in your repo named `components/alerts.jsonnet`:

```
local k = import "k.libsonnet";

local alerts = {
  kind: 'PrometheusRule',
  apiVersion: 'monitoring.coreos.com/v1',
  metadata: {
    name: 'k8s-demo',
    labels: {
      role: 'alert-rules',
      prometheus: 'k8s',
    },
  },
  spec: {
    groups: [
      {
        name: 'k8s-demo.rules',
        rules: [
          {
            alert: 'ElevatedLatency',
            expr: 'histogram_quantile(0.98, sum(increase(k8s_demo_rest_api_histogram_seconds_bucket[5m]))\n  by (service_name, le, code, path, method)) > 3\n',
            'for': '4m',
            labels: {
              severity: 'warning',
            },
          },
          {
            alert: 'ElevatedLatency',
            expr: 'histogram_quantile(0.98, sum(increase(k8s_demo_rest_api_histogram_seconds_bucket[5m]))\n  by (service_name, le, code, path, method)) > 7\n',
            'for': '4m',
            labels: {
              severity: 'critical',
            },
          },
        ],
      },
    ],
  },
};

k.core.v1.list.new([alerts])
```

Then you'd do a show and deploy via:

```
ks show minikube -c alerts
ks apply minikube -c alerts
```

Within a couple of minutes you'll find your new alerts deployed to
which ever cluster you deployed to. The intention would be to run this
via your CICD pipeline. You can also deploy this locally via Minikube.

## Visibility in AWS

Once deployed to a cluster you can view your alerts via the
[prometheus] endpoint in that cluster, for example:

- https://prometheus.playground1.dev.us-west-2.splunk8s.io/rules

## Considerations

- Currently alerts live in the same Kubernetes namespace as the
  service itself. We still need to sort out implications where users
  deploy the same service into their own namespace. This is because
  the result would be the same logical alert specified duplicated by
  multiple namespaces.
- This document does not describe how alerts get routed. TODO.

[//]: <> (References)

[alert crd]: example/prometheus-operator-crd/prometheusrule.crd.yaml
[alerts]: https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/
[crd]: https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/
[design]: https://docs.google.com/document/d/1V5pSP_b3Q7j79-IByr1_p77LRcjGHszkUu0lO09Homs/edit?usp=sharing
[implementation]: https://github.com/coreos/prometheus-operator/pull/1333
[prometheus]: https://prometheus.io/
[prometheus operator]: https://github.com/coreos/prometheus-operator
[promql]: https://prometheus.io/docs/prometheus/latest/querying/basics/
