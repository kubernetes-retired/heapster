package elasticsearch

const mapping = `{
  "aliases": {
    "heapster-events": {

    }
  },
  "mappings": {
    "k8s-heapster": {
      "properties": {
        "MetricsName": {
          "type": "string",
          "index": "analyzed",
          "fields": {
            "raw": {
              "type": "string",
              "index": "not_analyzed"
            }
          }
        },
        "MetricsTags": {
          "properties": {
            "container_base_image": {
              "type": "string",
              "index": "analyzed",
              "fields": {
                "raw": {
                  "type": "string",
                  "index": "not_analyzed"
                }
              }
            },
            "container_name": {
              "type": "string",
              "index": "analyzed",
              "fields": {
                "raw": {
                  "type": "string",
                  "index": "not_analyzed"
                }
              }
            },
            "host_id": {
              "type": "string",
              "index": "not_analyzed"
            },
            "hostname": {
              "type": "string",
              "index": "analyzed",
              "fields": {
                "raw": {
                  "type": "string",
                  "index": "not_analyzed"
                }
              }
            },
            "labels": {
              "type": "string",
              "index": "analyzed",
              "fields": {
                "raw": {
                  "type": "string",
                  "index": "not_analyzed"
                }
              }
            },
            "namespace_id": {
              "type": "string",
              "index": "not_analyzed"
            },
            "namespace_name": {
              "type": "string",
              "index": "not_analyzed"
            },
            "nodename": {
              "type": "string",
              "index": "analyzed",
              "fields": {
                "raw": {
                  "type": "string",
                  "index": "not_analyzed"
                }
              }
            },
            "pod_id": {
              "type": "string",
              "index": "not_analyzed"
            },
            "pod_name": {
              "type": "string",
              "index": "analyzed",
              "fields": {
                "raw": {
                  "type": "string",
                  "index": "not_analyzed"
                }
              }
            },
            "pod_namespace": {
              "type": "string",
              "index": "not_analyzed"
            },
            "resource_id": {
              "type": "string",
              "index": "not_analyzed"
            },
            "type": {
              "type": "string",
              "index": "not_analyzed"
            }
          }
        },
        "MetricsTimestamp": {
          "type": "date",
          "format": "strict_date_optional_time||epoch_millis"
        },
        "MetricsValue": {
          "properties": {
            "value": {
              "type": "long"
            }
          }
        }
      }
    },
    "events": {
      "properties": {
        "EventTags": {
          "properties": {
            "eventID": {
              "type": "string",
              "index": "not_analyzed"
            },
            "hostname": {
              "type": "string",
              "index": "analyzed",
              "fields": {
                "raw": {
                  "type": "string",
                  "index": "not_analyzed"
                }
              }
            },
            "pod_id": {
              "type": "string",
              "index": "not_analyzed"
            },
            "pod_name": {
              "type": "string",
              "index": "analyzed",
              "fields": {
                "raw": {
                  "type": "string",
                  "index": "not_analyzed"
                }
              }
            }
          }
        },
        "EventTimestamp": {
          "type": "date",
          "format": "strict_date_optional_time||epoch_millis"
        },
        "EventValue": {
          "type": "string"
        }
      }
    }
  }
}`
