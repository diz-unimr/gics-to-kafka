FROM golang:1.20.4-alpine3.16 AS build

RUN set -ex && \
    apk add --no-progress --no-cache \
        gcc=11.2.1_git20220219-r2 \
        musl-dev=1.2.3-r3

WORKDIR /app
COPY go.* ./
RUN go mod download

COPY . .
RUN go get -d -v && GOOS=linux GOARCH=amd64 go build -v -tags musl

FROM alpine:3.18 as run

RUN apk add --no-progress --no-cache tzdata=2023c-r0

ENV UID=65532
ENV GID=65532
ENV USER=nonroot
ENV GROUP=nonroot

RUN addgroup -g $GID $GROUP && \
    adduser --shell /sbin/nologin --disabled-password \
    --no-create-home --uid $UID --ingroup $GROUP $USER

WORKDIR /app/
COPY --from=build /app/gics-to-kafka /app/app.yml ./
USER $USER

ENV GIN_MODE=release
EXPOSE 8080

HEALTHCHECK --interval=1m CMD wget -q --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/gics-to-kafka"]
