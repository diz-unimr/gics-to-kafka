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
COPY --from=build /app/fhir-to-server .
COPY --from=build /app/app.yml .
ENTRYPOINT ["/app/gics-to-fhir"]
