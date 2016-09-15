FROM golang:1.7

ENV GOPATH /go

RUN curl https://glide.sh/get | sh
