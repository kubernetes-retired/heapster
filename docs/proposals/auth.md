Heapster Authentication and Authorization
=========================================

Current State
-------------

Heapster authentication is based on checking client certificates against a CA.
Authorization is either just a static list of names, or non-existant.  In
either case, anyone authenticated is authorized to perform any operation.  With
push metrics, the proposed plan was to allow adding a second CA, to allow
separating authorization for model and Oldtimer queries from authorization to
push metrics.  However, this is suboptimal, since it also separates
authentication.

Requirements
------------

Heapster should be able to separate authentication from authorization, and
the authorization interface should be flexible enough to support authorization
implementations that are able to make the following distinctions:

- containing queries to a particular namespace: we should be able to only allow
  pods to query for metrics within their own namespace.

- separating querying from pushing: we should allow certain users to push, and
  certain users to query, but not necessarily tie the two together

- allowing different push scope permissions: we should be able to specify that
  some users are only allowed to push metrics for the pods within their own
  namespace (or given sufficient information in the future, possibly their own
  RC), while others should be allowed to push for any object in the cluster.

Proposed Solution
-----------------

Heapster will adopt an authentication and authorization model based around
querying the Kubernetes model.  Authentication will support either client
certificate and tokens using the Kubernetes API server tokenreview
endpoint. Authorization will be done using the Kubernetes API server
subjectaccessreviews endpoint.

### Authentication ###

Authentication will support both client certificates and tokens.  Client
certificates will be supported by checking against a configurable CA (this
will generally be the cluster CA that gets injected into the Heapster
pod), while tokens will be checked against the Kubernetes API server using
a `TokenReview`.

The configuration will look as such:

- `--authn-ca=/path/to/ca.crt`
- `--auth-apiserver=https://$KUBE`

### Authorization ###

Authorization will work by matching Heapster concepts to Kubernetes RBAC
concepts, and then performing `SubjectAccessReview` requests against the
API server configured in `--auth-apiserver`.

Concepts will map as such:

#### Historical Queries #####

**Verb**: get / **Subresource**: historical-metrics

Namespace and Pod requests map to their respective resources.  Requests to
get container metrics would map a request to get metrics on the
corresponding pod.  Requests for node and system container metrics would
simply map to cluster permissions to get historical-metrics on nodes.

Getting by Pod UID would require permissions to get historical metrics on
pods in any namespace, since we cannot tell namespace from the pod UID.

Geting pods by a list would require making multiple requests, although for
the sake of performance, it may be desirable to just merge them into
`namespace=$NS, resource=pod, name=""` request.

#### Model Queries ####

**Verb**: get / **Subresource**: metrics

These function more or less identically to historical queries, except that
they don't have to worry about Pod UIDs.

#### Push Creation ####

**Verb**: create / **Subresource**: metrics, unprefixed-metrics

The metrics subresource indicates that a user is allowed to push metrics
not prefixed with their username.  The unprefixed-metrics subresource
indicates that a user is allowed to push metrics with no prefix.  Core
system metrics can never be overriden, however.

In order to determine the resources to use for push metrics, the push
handler will first determine the full set of resources involved, and will
then aggregate according to the following rules:

- a request containing only metrics for a single object (e.g. pods,
  services, etc) will turn into a query against that resource (containers
  in a pod are considered equivalent to the pod here).

- a request involving multiple objects of a given type (e.g. multiple
  pods) in a namespace will yield a review for `namespace=$NS,
  resource=$RES, name=""` for that resource.

- metrics describing a namespace will yield a review against that
  namespace.

- metrics describing nodes or system containers on a node will yield
  a review against each node in question.

Additionally it may be valuable to simply detect when a large number of
namespaces are in play, and combine that all into a single cluster-wide
review against `namespace="*"` (for cases when a "system" component is
pushing metrics against all namespaces).

Implementation
--------------

The authenticator would be inserted as a filter in the go-restful handlers,
which would then inject authentication information into the request (via
go-restful request attributes).  Then, each of the different handlers would
be responsible for fetching the authenticator's user info from the attribute,
invoking the authorizer, and terminating the request if the authorizer returns
an error.  For instance, the model handlers can simply construct the needed
information from the route URL, while the push handler might extract the metric
names, keys, and prefix, and then call the authorizer with that information.
