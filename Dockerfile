FROM golang:1.19.3-alpine3.16 AS build

RUN set -ex && \
    apk add --no-progress --no-cache \
        gcc \
        musl-dev

WORKDIR /app
COPY go.* ./
RUN go mod download

COPY . .
RUN go get -d -v
RUN GOOS=linux GOARCH=amd64 go build -v -tags musl

FROM alpine:3.16 as run

RUN apk add --no-progress --no-cache tzdata

WORKDIR /app/
COPY --from=build /app/gics-to-kafka .
COPY --from=build /app/app.yml .
ENV GIN_MODE=release

ENTRYPOINT ["/app/gics-to-kafka"]
