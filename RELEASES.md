# Release Notes

## 0.7 (2-18-2014)
- Support for Google Cloud Monitoring Backend
- Watch kubernetes api-server instead of polling for pods and nodes info.
- Fetch stats in parallel.
- Refactor code and improve testing.
- Miscellaneous bug fixes.
- Native support for CoreOS.

## 0.6 (1-21-2014)
- New /validate REST endpoint to probe heapster.
- Heapster supports kube namespaces.
- Heapster uses InfluxDB service DNS name while running in kube mode.

## 0.5 (12-11-2014)
- Compatiblity with updated InfluxDB service names.

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