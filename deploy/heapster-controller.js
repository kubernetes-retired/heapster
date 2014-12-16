{
  "id": "monitoring-heapsterController",
  "kind": "ReplicationController",
  "apiVersion": "v1beta1",
  "desiredState": {
    "replicas": 1,
    "replicaSelector": {"name": "heapster"},
    "podTemplate": {
      "desiredState": {
         "manifest": {
           "version": "v1beta1",
           "id": "monitoring-heapsterController",
           "containers": [{
             "name": "heapster",
             "image": "kubernetes/heapster:canary",
           }]
         }
      },
      "labels": {
        "name": "heapster",
        "uses": "monitoring-influxdb"
      }
    }
  },
  "labels": {"name": "heapster"}
}
