FROM golang:1.18 AS build

ARG RELEASE_STRING=dev
ENV IMPORT_PATH="github.com/plumber-cd/terraform-backend-git/cmd"
WORKDIR /go/delivery
COPY . .
RUN mkdir bin && go build \
    -ldflags "-X ${IMPORT_PATH}.Version=${RELEASE_STRING}" \
    -o ./bin ./...

FROM debian:buster

COPY --from=build /go/delivery/bin /usr/bin
