{
  "apiVersion": "v1beta3",
  "kind": "ReplicationController",
  "metadata": {
    "labels": {
      "name": "heapster",
      "kubernetes.io/cluster-service": "true"
    },
    "name": "monitoring-heapster-controller"
  },
  "spec": {
    "replicas": 1,
    "selector": {
      "name": "heapster"
    },
    "template": {
      "metadata": {
        "labels": {
          "name": "heapster",
          "kubernetes.io/cluster-service": "true"
        }
      },
      "spec": {
        "containers": [
          {
            "image": "kubernetes/heapster:v0.10.0",
            "name": "heapster",
            "env": [
              {
                "name": "INFLUXDB_HOST",
                "value": "monitoring-influxdb"
              },
              {
                "name": "SINK",
                "value": "influxdb"
              },
              {
                "name": "FLAGS",
                "value": "--kubernetes_version=v1beta3"
              }
            ],
            "volumeMounts": [
              {
                "name": "ssl-certs",
                "mountPath": "/etc/ssl/certs",
                "readOnly": true
              }
            ]
          }
        ],
        "volumes": [
          {
            "name": "ssl-certs",
            "source": {
              "hostDir": {
                "path": "/etc/ssl/certs"
              }
            }
          }
        ]
      }
    }
  }
}