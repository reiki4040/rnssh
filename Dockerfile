FROM golang:1.10

ENV GOPATH /go

RUN curl https://glide.sh/get | sh
