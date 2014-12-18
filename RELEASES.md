# Release Notes

## 0.5 (18-02-2014)
- Compatibility with Kubernetes v0.7.0

## 0.4 (12-02-2014)
- Compatibility with cAdvisor v0.6.2

## 0.3 (11-26-2014)
- Handle updated Pod API in Kubernetes.

## 0.2 (10-06-2014)
- Use kubernetes master readonly service which does not require auth

## 0.1 (10-05-2014)
- First version of heapster.
- Native support for kubernetes and CoreOS.
- For Kubernetes gets pods and rootcgroup information.
- For CoreOS gets containers and rootcgroup information.
- Supports InfluxDB and bigquery.
- Exports pods and container stats in table 'stats' in InfluxDB
- rootCgroup is exported in table 'machine' in InfluxDB
- Special dashboard for kubernetes.