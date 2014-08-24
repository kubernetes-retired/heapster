tutum-docker-grafana
====================

Grafana dashboard for Influx DB


Usage
-----
To create the image `tutum/grafana`, execute the following command on the tutum-docker-grafana folder:

    docker build -t tutum/grafana .

To run the image and bind the port:

    docker run -d -p 80:80 tutum/grafana
    
The first time that you run your container, a new user `admin` will be created for HTTP basic auth with a random password. To get the password, check the logs of the container by running:

    docker logs <CONTAINER_ID>

You will see an output like the following:
```
 ========================================================================
 You can now connect to Grafana with the following credential:
 
     admin:ilNfrVn68r1N

 ========================================================================
```
In this case, `ilNfrVn68r1N` is the password allocated to the `admin` user.

You can now login you to Grafana in your browser: `http://127.0.0.1/`


Setting a specific password for Basic HTTP Authentication
---------------------------------------------------------

You can specify username and password for HTTP Basic Auth of `tutum/grafana`:

```
HTTP_USER=admin                 Username for HTTP auth, using admin by default
HTTP_PASS=**Random**        Password for HTTP auth. Change it, otherwise system will generate a random password.
```

If you want to user a preset password instead of a random generated one, you can set the environment variable `HTTP_PASS` to you specific password when running the container:

    docker run -d -p 80:80 -e HTTP_USER=admin -e HTTP_PASS=mypass tutum/grafana

You can now test it: `http://127.0.0.1/`


Configure the connection to InfluxDB
------------------------------------

`tutum/grafana` needs to know the information of your InfluxDB for configuration. Please provide the following environment variables when running your Grafana container:
```
INFLUXDB_PROTO=http                 Protocol of your InfluxDB
INFLUXDB_HOST=**ChangeMe**          Host of your InfluxDB (without protocol)
INFLUXDB_PORT=8086                  Port number of your InfluxDB
INFLUXDB_NAME=**ChangeMe**          Database name of your InfluxDB
INFLUXDB_USER=root                  Username of your InfluxDB
INFLUXDB_PASS=root                  Password of your InfluxDB
```

Here is an example:

    docker run -d -p 80:80 -e INFLUXDB_HOST=influxdb-1-tifayuki.delta.tutum.io -e INFLUXDB_PORT=8086 -e INFLUXDB_NAME=testdb -e INFLUXDB_USER=root -e INFLUXDB_PASS=root tutum/grafana


Configure Elasticsearch to save and load dashboards
---------------------------------------------------
If you want you use Elasticsearch to save and load you dashboards, you can provide the following environment variables for configuration:

```
ELASTICSEARCH_PROTO=http            Protocol of your Elasticsearch
ELASTICSEARCH_HOST=**None**         Host of your Elasticsearch (without protocol)
ELASTICSEARCH_PORT=9200             Port number of your Elasticsearch
ELASTICSEARCH_USER=**None**         Username for elasticsearch if it has HTTP basic auth enabled (leave it to **None** if no HTTP basic auth is needed)
ELASTICSEARCH_PASS=**None**         Password for elasticsearch if it has HTTP basic auth enabled (leave it to **None** if no HTTP basic auth is needed)
```

Here is an example:

    docker run -d -p 80:80 -e INFLUXDB_HOST=influxdb-1-tifayuki.delta.tutum.io -e INFLUXDB_PORT=8086 -e INFLUXDB_NAME=testdb -e INFLUXDB_USER=root -e INFLUXDB_PASS=root -e ELASTICSEARCH_HOST=elasticsearch-1-tifayuki.beta.tutum.io -e ELASTICSEARCH_PORT=9200 -e ELASTICSEARCH_USER=admin -e ELASTICSEARCH_PASS=admin tutum/grafana


**by http://www.tutum.co**
