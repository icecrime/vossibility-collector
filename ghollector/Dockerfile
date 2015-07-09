FROM golang:1.4.2
MAINTAINER Arnaud Porterie <icecrime@docker.com>

RUN go get github.com/BurntSushi/toml
RUN go get github.com/Sirupsen/logrus
RUN go get github.com/bitly/go-nsq
RUN go get github.com/bitly/go-simplejson
RUN go get github.com/codegangsta/cli
RUN go get github.com/google/go-github/github
RUN go get github.com/mattbaird/elastigo && (cd /go/src/github.com/mattbaird/elastigo && git checkout v1.0)
RUN go get golang.org/x/oauth2

ADD . /go/src/github.com/icecrime/vossibility/ghollector/
RUN go install github.com/icecrime/vossibility/ghollector/

ENTRYPOINT ["ghollector"]
