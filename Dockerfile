FROM golang:1.4.2
MAINTAINER Arnaud Porterie <icecrime@docker.com>

# Install GB dependency manager
RUN go get github.com/constabulary/gb/...

# Build the project
ADD . /src
WORKDIR /src
RUN gb build all

# Set the entrypoint
ENTRYPOINT ["/src/bin/vossibility-collector"]
