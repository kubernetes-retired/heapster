# Statster

Statster is a fork of K8's [Heapster](https://github.com/kubernetes/heapster) focusing on providing high level cluster materics that can be feed into an external monitoring/alerting platform (Graphite).
Unlike heapster that focuses on incluster per-pod stats statster exports data at higher levels (deployments, daemonsets, service, etc...)

Statster currently supports [Kubernetes](https://github.com/kubernetes/kubernetes).

### Running Statster on Kubernetes

To run Heapster on a Kubernetes cluster with,
- Graphite/Grafatan via graphite protocol.
- Librator hosted service.
- AWS Cloudwatch (TODO)

## Community

Contributions, questions, and comments are all welcomed and encouraged!
