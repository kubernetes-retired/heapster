# integration

Integration with kafka & zoopeeker test helpers.

`KafkaCluster` depends on `docker` and `docker-compose` commands.

**IMPORTANT**: Make sure to update `KAFKA_ADVERTISED_HOST_NAME` in
`kafka-docker/docker-compose.yml` before running tests.

## kafka-docker

[kafka-docker](/integration/kafka-docker) directory is copy of
[wurstmeister/kafka-docker](https://github.com/wurstmeister/kafka-docker)
repository.
