# gics-to-kafka
[![MegaLinter](https://github.com/diz-unimr/gics-to-kafka/workflows/MegaLinter/badge.svg?branch=main)](https://github.com/diz-unimr/gics-to-kafka/actions?query=workflow%3AMegaLinter+branch%3Amain) ![go](https://github.com/diz-unimr/gics-to-kafka/actions/workflows/build.yml/badge.svg) ![docker](https://github.com/diz-unimr/gics-to-kafka/actions/workflows/release.yml/badge.svg) [![codecov](https://codecov.io/gh/diz-unimr/gics-to-kafka/branch/main/graph/badge.svg?token=D66XMZ5ALR)](https://codecov.io/gh/diz-unimr/gics-to-kafka)
> Receive gICS notifications and send them to a Kafka topic

This is a minimal Kafka producer which exposes a single HTTP endpoint to receive messages via 
the gICS notifications service (connectionType: HTTP) and sends them to a Kafka topic.

## HTTP Endpoint

The `POST` endpoint for receiving notifications is: `/notification`

## Configuration properties

| Name                             | Default                | Description                             |
|----------------------------------|------------------------|-----------------------------------------|
| `app.name`                       | gics-to-kafka          | Application name                        |
| `app.log-level`                  | info                   | Log level (error,warn,info,debug,trace) |
| `app.http.auth.user`             |                        | HTTP endpoint Basic Auth user           |
| `app.http.auth.password`         |                        | HTTP endpoint Basic Auth password       |
| `app.http.port`                  | 8080                   | HTTP endpoint port                      |
| `gics.signer-id`                 | Patienten-ID           | Target consent signerId                 |
| `kafka.bootstrap-servers`        | localhost:9092         | Kafka brokers                           |
| `kafka.security-protocol`        | ssl                    | Kafka communication protocol            |
| `kafka.output-topic`             | gics-notification      | Kafka topic to produce to               |
| `kafka.ssl.ca-location`          | /app/cert/kafka-ca.pem | Kafka CA certificate location           |
| `kafka.ssl.certificate-location` | /app/cert/app-cert.pem | Client certificate location             |
| `kafka.ssl.key-location`         | /app/cert/app-key.pem  | Client key location                     |
| `kafka.ssl.key-password`         | private-key-password   | Client key password                     |

### Environment variables

Override configuration properties by providing environment variables with their respective names.
Upper case env variables are supported as well as underscores (`_`) instead of `.` and `-`.

## Configure gICS

In order to receive notifications from gICS, the notification service has to be enabled and configured accordingly.
The following SQL script can be used to update the `configuration` table in the `notification_service` database:

```sql
USE `notification_service`;
UPDATE `configuration` SET `value` = 
'<org.emau.icmvc.ttp.notification.service.model.config.NotificationConfig>
    <consumerConfigs>
        <param>
            <key>HTTPConsumer</key>
            <value class="org.emau.icmvc.ttp.notification.service.model.config.ConsumerConfig">
                <connectionType>HTTP</connectionType>
                <messageTypes>
                    <string>*</string>
                </messageTypes>
                <excludeClientIdFilter
                    class="set">
                    <string>E-PIX_Web</string>
                </excludeClientIdFilter>
                <parameter>
                    <param>
                        <key>url</key>
                        <value>http://gics-to-kafka:8080/notification</value>
                    </param>
                    <param>
                        <key>username</key>
                        <value>test</value>
                    </param>
                    <param>
                        <key>password</key>
                        <value>test</value>
                    </param>
                </parameter>
            </value>
        </param>
    </consumerConfigs>
</org.emau.icmvc.ttp.notification.service.model.config.NotificationConfig>'
WHERE `configKey` = 'notification.config';
```

# Deployment

Example via `docker compose`:
```yml
gics-to-kafka:
    image: ghcr.io/diz-unimr/gics-to-kafka:latest
    restart: unless-stopped
    environment:
      APP_NAME: gics-to-kafka
      APP_HTTP_AUTH_USER: test
      APP_HTTP_AUTH_PASSWORD: test
      APP_HTTP_PORT: 8080
      APP_LOG_LEVEL: info
      GICS_SIGNER_ID: Patienten-ID
      KAFKA_BOOTSTRAP_SERVERS: kafka:19092
      KAFKA_SECURITY_PROTOCOL: SSL
      KAFKA_OUTPUT_TOPIC: gics-notification
    volumes:
     - ./cert/ca-cert:/app/cert/kafka-ca.pem:ro
     - ./cert/gics-to-kafka.pem:/app/cert/app-cert.pem:ro
     - ./cert/gics-to-kafka.key:/app/cert/app-key.pem:ro
```

# License

[AGPL-3.0](https://www.gnu.org/licenses/agpl-3.0.en.html)