FROM golang

RUN apt-get update
RUN apt-get dist-upgrade -y
RUN apt-get install -y \
	build-essential pkg-config \
	libpth-dev \
	libavcodec-dev libavutil-dev libavformat-dev libswscale-dev

WORKDIR /app

# Try to cache deps
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
