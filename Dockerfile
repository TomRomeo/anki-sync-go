# syntax=docker/dockerfile:1

FROM golang:1.17.6-alpine

WORKDIR /build

# We need gcc installed
RUN apk add build-base

COPY go.mod go.sum ./

RUN go mod graph | awk '{if ($1 !~ "@") print $2}' | xargs go get

COPY . .

RUN go build -o /ankisyncd

ENTRYPOINT [ "/ankisyncd" ]
