# Release Notes

## 0.7 (4-28-2015)
- Updated kubernetes dashboard to have a grouping time of 5s.

## 0.6 (3-30-2015)
- Switch to kuisp for serving files & proxying to services
- Configurable reverse proxy directly to InfluxDB service rather than using
apiserver proxy

## 0.5 (3-13-2015)
- Default dashboard uses the metrics schema exported by heapster version >= v0.9
- Grafana container works on bare metal kubernetes setups.

## 0.4 (2-2-2015)
- Make Grafana image work with kubernetes apiserver proxy service.
