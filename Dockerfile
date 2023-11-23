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
RUN GO_VERSION=$(curl -sSL "https://go.dev/VERSION?m=text" | awk 'NR==1{print $1}') && \
    curl -sSL "https://dl.google.com/go/${GO_VERSION}.linux-amd64.tar.gz" | tar xpz -C /usr/local
ENV PATH=$PATH:/usr/local/go/bin

# Try to cache deps
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
