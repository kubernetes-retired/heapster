{
  "id": "monitoring-influxGrafanaController",
  "kind": "ReplicationController",
  "apiVersion": "v1beta1",
  "desiredState": {
    "replicas": 1,
    "replicaSelector": {"name": "influxGrafana"},
    "podTemplate": {
      "desiredState": {
         "manifest": {
           "version": "v1beta1",
           "id": "monitoring-influxGrafanaController",
           "containers": [{
	       "name": "influxdb",
	       "image": "kubernetes/heapster_influxdb:v0.2",
	       "ports": [{"containerPort": 8083, "hostPort": 8083},
	                 {"containerPort": 8086, "hostPort": 8086},
			 {"containerPort": 8090, "hostPort": 8090},
			 {"containerPort": 8099, "hostPort": 8099}]
	    }, {
	       "name": "grafana",
	       "image": "kubernetes/heapster_grafana:v0.2",
	       "ports": [{"containerPort": 80, "hostPort": 80}],
	       "env": [{"name": HTTP_USER, "value": "admin"},
	       	       {"name": HTTP_PASS, "value": "**None**"}]
	    }]
	 }
      },
      "labels": {
        "name": "influxGrafana",
      }
    }
  },
  "labels": {"name": "influxGrafana"}
}
