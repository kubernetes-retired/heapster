// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
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
