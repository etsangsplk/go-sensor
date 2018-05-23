FROM golang:1.10 as build-stage

# Set WORKDIR to project path in GOPATH
WORKDIR /go/src/github.com/splunk/ssc-observation

# Copy repository
COPY . /go/src/github.com/splunk/ssc-observation/

RUN go get github.com/golang/dep/cmd/dep

RUN CGO_ENABLED=0 dep ensure
