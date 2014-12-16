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
	       "image": "kubernetes/heapster_influxdb",
	       "ports": [{"containerPort": 8083, "hostPort": 8083},
	                 {"containerPort": 8086, "hostPort": 8086},
			 {"containerPort": 8090, "hostPort": 8090},
			 {"containerPort": 8099, "hostPort": 8099}]
	    }, {
	       "name": "grafana",
	       "image": "kubernetes/heapster_grafana:canary",
	       "ports": [{"containerPort": 80, "hostPort": 80}],
	       "env": [{"name": HTTP_USER, "value": "admin"},
	       	       {"name": HTTP_PASS, "value": "**None**"}]
	    }, {
	    	"name": "elasticsearch",
		"image": "dockerfile/elasticsearch",
		"ports": [{"containerPort": 9200, "hostPort": 9200},
			  {"containerPort": 9300, "hostPort": 9300}],
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
