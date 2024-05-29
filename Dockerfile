FROM golang:1.22-alpine3.19 AS build

RUN set -ex && \
    apk add --no-progress --no-cache \
        gcc \
        musl-dev

WORKDIR /app
COPY go.* ./
RUN go mod download

COPY . .
RUN go get -d -v && GOOS=linux GOARCH=amd64 go build -v -tags musl

FROM alpine:3.20 as run

RUN apk add --no-progress --no-cache tzdata

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

HEALTHCHECK --interval=1m CMD wget -q --tries=1 http://localhost:8080/health -O /dev/null || exit 1

ENTRYPOINT ["/app/gics-to-kafka"]
