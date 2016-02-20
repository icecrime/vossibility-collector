FROM golang:1.6.0-alpine
MAINTAINER Arnaud Porterie <icecrime@docker.com>

# Install git
RUN apk update && apk add git

# Install GB dependency manager
RUN go get github.com/constabulary/gb/...

# Build the project
ADD . /src
WORKDIR /src
RUN gb build all

# Set the entrypoint
ENTRYPOINT ["/src/bin/vossibility-collector"]
