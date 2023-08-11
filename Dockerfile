FROM golang:1.19.3-alpine3.16 AS build

RUN set -ex && \
    apk add --no-progress --no-cache \
        gcc \
        musl-dev

WORKDIR /app
COPY go.* ./
RUN go mod download

COPY . .
RUN go get -d -v && GOOS=linux GOARCH=amd64 go build -v -tags musl

FROM alpine:3.16 as run

RUN apk add --no-progress --no-cache tzdata

ENV UID=65532
ENV GID=65532
ENV USER=nonroot
ENV GROUP=nonroot

RUN addgroup -g $GID $GROUP && \
    adduser --shell /sbin/nologin --disabled-password \
    --no-create-home --uid $UID --ingroup $GROUP $USER

WORKDIR /app/
COPY --from=build --chown=$USER:$USER /app/gics-to-kafka .
COPY --from=build --chown=$USER:$USER /app/app.yml .
USER $USER

ENV GIN_MODE=release
EXPOSE 8080

HEALTHCHECK --interval=1m CMD wget -q --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/gics-to-kafka"]
