# Heapster Deprecation Timeline

This is the (proposed) timeline for Heapster deprecation.  Any changes
made to the timeline will be reflected here.  Note that this is the
timeline for the official Heapster repository.  Individual distributions
are encouraged to follow suit in deprecating Heapster, but may continue to
support it on their own.

## Summary

| Kubernetes Release  | Action              | Policy/Support                                                                   |
|---------------------|---------------------|----------------------------------------------------------------------------------|
| Kubernetes 1.11     | Initial Deprecation | No new features or sinks are added.  Bugfixes may be made.                       |
| Kubernetes 1.12     | Setup Removal       | The optional to install Heapster via the Kubernetes setup script is removed.     |
| Kubernetes 1.13     | Removal             | No new bugfixes will be made.  Move to kubernetes-retired organization.          |

## Milestones

### Initial Deprecation (Kubernetes 1.11)

Heapster is marked deprecated as of Kubernetes 1.11.  Users are encouraged
to use
[metrics-server](https://github.com/kubernetes-incubator/metrics-server)
instead, potentially supplemented by a third-party monitoring solution,
such as Prometheus.

No new features will be added (including, but not limitted to, new sinks,
enhancements to existing sinks, enhancements to sources, and enhancements
to processing).

The Heapster maintainers will continue to accept bugfixes for critical
bugs.  People filing bugs will be encouraged to reproduce with
metrics-server, and file there.

Similarly, **the legacy HPA clients and setup flags in Kubernetes should
be marked as deprecated.**

### Setup Removal (Kubernetes 1.12)

Support for installing Heapster will be removed from the Kubernetes setup
scripts.

Additional warnings will be made in the Kubernetes release notes.

### Removal (Kubernetes 1.13)

Heapster will be migrated to the kubernetes-retired organization.  No new
code will be merged by the Heapster maintainers.
