# Table of Contents

To initiaze an instana tracer properly, you need to set some environment varibles.

# Environment Variables

Note if INSTANA_AGENT_HOST and INSTANA_AGENT_PORT are not set, it will use default INSTANA settings and send to their public
endpoint. For default settings, you don't need to set an environmen variables.
INSTANA_AGENT_HOST=saas-us-west-2.instana.io
INSTANA_AGENT_PORT=42699
are the default values


(Optional) variables will be set to LightStep default values. Refer to instana documentation for default values.

| Envrionment variable       |  Format |  Comment  | Example |
| :------------- |:-------------:| :----- | :-----: |
| INSTANA_AGENT_HOST |  host | (Required) agent host | 127.0.0.1 |
| INSTANA_AGENT_PORT |  port | (Required) agent port | 8080 |
| INSTANA_MAXBUFFERED_SPANS | integer as string  |  (Optional) maximum number of spans that will be buffered  | 10 |
| INSTANA_FORCED_TRANSMISSION_AT |  integer as string   |  (Optional) force sending to agent when number of spans reached this number  | 100 |
| INSTANA_LOG_LEVEL | integer |  (Optional) minimal log level for tracer to send  | Error = 0 .. Debug =3 |

To test your application locally and with a locally installed agent.
Using go-sql as example.

1) bring up your instana agent 
The following connects to the default public INSTANA Agent (sass-us-west-2.instana.io). 

```
sudo docker run   --detach   --name instana-agent   --volume /var/run/docker.sock:/var/run/docker.sock   --volume /dev:/dev   --volume /sys:/sys   --volume /var/log:/var/log   --privileged   --net=host   --pid=host   --ipc=host   --env="INSTANA_AGENT_KEY=ASK_YOUR_LEAD"   --env="INSTANA_AGENT_ENDPOINT=saas-us-west-2.instana.io"   --env="INSTANA_AGENT_ENDPOINT_PORT=443"  -p 443:443  -p 42699:42699 instana/agent
```


2) You need your applicaiton to be dockerized if you are using OSX. (that' instana agent requirement)
And run your sqlx-example.

``
Create your container for your golang app

1) make sqlx-example-docker

Run it 

2) docker run splunk/sqlx-example splunk/sqlx-example

```

3) Checking if your INSTANA AGENT is accepting events from your golang app

 ```
docker logs instana-agent
 ```

Check that your go-sensor is activate 

 ```
 Starting Instana Agent ...
Starting Instana Agent ...
2018-10-25T18:10:44.838+00:00 | INFO  | agent-starter                    | turesManagerImpl | com.instana.agent-bootstrap - 1.1.11 | Installed instana-activemq-discovery/1.1.8
2018-10-25T18:10:44.841+00:00 | INFO  | agent-starter                    | turesManagerImpl |
.....
com.instana.agent-bootstrap - 1.1.11 | Installed instana-glassfish-discovery/1.1.8
2018-10-25T18:10:44.847+00:00 | INFO  | agent-starter                    | turesManagerImpl | com.instana.agent-bootstrap - 1.1.11 | Installed instana-golang-discovery/1.2.6
2018-10-25T18:10:44.847+00:00 | INFO  | agent-starter                    | turesManagerImpl | 
 ```

 Check that go-sensor is receving events frpm your golang-app. While your golang app is sending events to agent, you should see logs like this

 ```
 com.instana.sensor-golang - 1.2.6 | Activated Sensor
2018-10-25T18:31:42.698+00:00 | INFO  | 84d-9cd3-4b88-8bb0-99b56c475494) | Process          | com.instana.sensor-process - 1.1.28 | Activated Sensor for PID 16261
2018-10-25T18:31:42.703+00:00 | INFO  | 84d-9cd3-4b88-8bb0-99b56c475494) | GolangTrace      | com.instana.sensor-golang-trace - 1.2.1 | Activated Sensor
2018-10-25T18:31:51.681+00:00 | INFO  | instana-scheduler-thread-3       | Process          | com.instana.sensor-process - 1.1.28 | Deactivated Sensor for PID 16261
2018-10-25T18:31:51.685+00:00 | INFO  | 84d-9cd3-4b88-8bb0-99b56c475494) | GolangTrace      | com.instana.sensor-golang-trace - 1.2.1 | Deactivated Sensor
2018-10-25T18:31:51.688+00:00 | INFO  | 84d-9cd3-4b88-8bb0-99b56c475494) | Golang           | com.instana.sensor-golang - 1.2.6 | Deactivated Sensor
2018-10-25T18:31:53.971+00:00 | INFO  | 01f1ebdcfb68c6bdf05acbfa2ab9adaf | Docker           | com.instana.sensor-docker - 1.1.49 | Deactivated Sensor
2018-10-25T18:37:22.611+00:00 | INFO  | bc0-7d09-4516-9099-d1c1f589da7c) | Docker           | com.instana.sensor-docker - 1.1.49 | Activated Sensor

 ```

Further examples:
Kvstore

1) Create local docker container images via:
``
make local-images
```

2) Running kvstore
Assume that your image is kvstore/kvservice:0.1.115-6-gfd6d80a

```

docker run -i -t -e KVSERVICE_ADDR="localhost:8066" -e KVSERVICE_DB_HOST="localhost" -e KVSERVICE_RATELIMIT_CALLSPERSECOND="1000" -e KVSERVICE_DB_READLIMIT_KBPERSECOND="1000000" -e KVSERVICE_DB_READBUCKETMAX_KB="30000000" -e KVSERVICE_DB_WRITEBUCKETMAX_KB="30000000" -e KVSERVICE_DB_STORAGELIMIT_MB="500"  -e KVSERVICE_MAX_THROTTLE_DURATION_SECONDS="30" -e KVSERVICE_DB_SIZECHECKINTERVAL_SECONDS="20" -e KVSERVICE_DEV_MODE="1" -e KVSERVICE_LOCAL_SECRETS="1" -e KVSERVICE_DB_WRITELIMIT_KBPERSECOND="5000000" -p 8066:8066 kvstore/kvservice:0.1.115-6-gfd6d80a kvstore/kvservice:0.1.115-6-gfd6d80a


```


Reference:

[instana tracer configuration](https://github.com/instana/go-sensor#opentracing)