FROM ubuntu:focal

WORKDIR /app

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update
RUN apt-get dist-upgrade -y
RUN apt-get install -y \
	build-essential \
	pkg-config \
	curl \
	libavcodec-dev \
	libavutil-dev \
	libavformat-dev \
	libswscale-dev

# Install Go
RUN curl -s \
	"https://dl.google.com/go/$(curl -s https://golang.org/VERSION?m=text).linux-amd64.tar.gz" \
	| tar xpz -C /usr/local
ENV PATH=$PATH:/usr/local/go/bin

# Try to cache deps
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
