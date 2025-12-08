FROM golang:1.24 AS build

ARG RELEASE_STRING=dev
ENV IMPORT_PATH="github.com/plumber-cd/terraform-backend-git/cmd"
WORKDIR /go/delivery
COPY . .
RUN mkdir bin && go build \
    -ldflags "-X ${IMPORT_PATH}.Version=${RELEASE_STRING}" \
    -o ./bin ./...

FROM debian:bookworm

# Include CA Certs to resolve TLS handshakes
RUN DEBIAN_FRONTEND="noninteractive" apt-get update && apt-get install -y ca-certificates && apt-get autoremove -y && apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

COPY --from=build /go/delivery/bin /usr/bin
