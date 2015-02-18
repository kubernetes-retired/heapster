Runnning Heapster on CoreOS
================================

Heapster enables cluster monitoring in a CoreOS cluster using [cAdvisor](https://github.com/google/cadvisor). 

NOTE: Some of the following steps should be handled using fleet and unit files.

**Step 1: Start cAdvisor on all hosts by default**

gce-cloud-config-with-cadvisor.yml contains 'cadvisor.service' which can be specified as part of your cloud config to bring up cAdvisor by default on all hosts.

**Step 2: Start InfluxDB**

On a CoreOS machine start InfluxDB and grafana

```shell
$ ../../influx-grafana/influxdb/start
```

**Step 3: Start heapster coreos buddy**

Heapster does not natively support CoreOS. To help with discovering all the hosts in the cluster start a heapster buddy container that will handle discovery.

```shell
$ mkdir heapster
$ docker run --name heapster-buddy --net host -d -v $PWD/heapster:/var/run/heapster vish/heapster-buddy-coreos
```

**Step 4: Start heapster**

Figure out the host where InfluxDB is running and pass that to heapster through 'INFLUXDB_HOST' environment variable.

```shell
$ docker run --name heapster -d -e INFLUXDB_HOST=<ip>:8086 -v $PWD/heapster:/var/run/heapster kubernetes/heapster
```

**Step 5: Start Grafana**

```
docker run -d -p 80:80 -e INFLUXDB_HOST=<host_ip> -e INFLUXDB_NAME=k8s tutum/grafana
```

cadvisor (globally deployed)
```
[Unit]
Description=cAdvisor Service
After=docker.service
Requires=docker.service

[Service]
TimeoutStartSec=10m
Restart=always
ExecStartPre=-/usr/bin/docker kill cadvisor
ExecStartPre=-/usr/bin/docker rm -f cadvisor
ExecStartPre=/usr/bin/docker pull google/cadvisor
ExecStart=/usr/bin/docker run --volume=/:/rootfs:ro --volume=/var/run:/var/run:rw --volume=/sys:/sys:ro --volume=/var/lib/docker/:/var/lib/docker:ro --publish=4194:4194 --name=cadvisor --net=host google/cadvisor:latest --logtostderr --port=4194
ExecStop=/usr/bin/docker stop -t 2 cadvisor

[X-Fleet]
Global=true
MachineMetadata=role=kubernetes
```

Influxdb 
```
[Unit]
Description=InfluxDB Service
After=docker.service
Requires=docker.service

[Service]
TimeoutStartSec=10m
Restart=always
ExecStartPre=-/usr/bin/docker kill influxdb
ExecStartPre=-/usr/bin/docker rm -f influxdb
ExecStartPre=/usr/bin/docker pull kubernetes/heapster_influxdb
ExecStart=/usr/bin/docker run --name influxdb -p 8083:8083 -p 8086:8086 -p 8090:8090 -p 8099:8099 kubernetes/heapster_influxdb
ExecStop=/usr/bin/docker stop -t 2 influxdb
```

Heapster agent (buddy)
```
[Unit]
Description=Heapster Agent Service
After=docker.service
Requires=docker.service

[Service]
TimeoutStartSec=10m
Restart=always
ExecStartPre=-/usr/bin/mkdir -p /home/core/heapster
ExecStartPre=-/usr/bin/docker kill heapster-agent
ExecStartPre=-/usr/bin/docker rm -f heapster-agent
ExecStartPre=/usr/bin/docker pull vish/heapster-buddy-coreos
ExecStart=/usr/bin/docker run --name heapster-agent --net host -v /home/core/heapster:/var/run/heapster vish/heapster-buddy-coreos
ExecStop=/usr/bin/docker stop -t 2 heapster-agent

[X-Fleet]
MachineOf=influxdb.service
```

Heapster
```
[Unit]
Description=Heapster Agent Service
After=docker.service
After=heapster-agent.service
Requires=docker.service
Requires=heapster-agent.service

[Service]
TimeoutStartSec=10m
Restart=always
ExecStartPre=-/usr/bin/docker kill heapster
ExecStartPre=-/usr/bin/docker rm -f heapster
ExecStartPre=/usr/bin/docker pull vish/heapster
ExecStart=/usr/bin/docker run --name heapster --net host -e INFLUXDB_HOST=127.0.0.1:8086 -v /home/core/heapster:/var/run/heapster vish/heapster
ExecStop=/usr/bin/docker stop -t 2 heapster

[X-Fleet]
MachineOf=heapster-agent.service

```
