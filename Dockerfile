FROM golang

RUN apt-get update
RUN apt-get dist-upgrade -y
RUN apt-get install -y build-essential pkg-config

# Compile newer FFmpeg and deps
RUN echo deb-src \
	http://ftp.debian.org/debian/ \
	stable main contrib non-free \
	>> /etc/apt/sources.list
RUN echo deb-src \
	http://ftp.debian.org/debian/ \
	stable-updates main contrib non-free \
	>> /etc/apt/sources.list
RUN echo deb-src \
	http://security.debian.org/debian-security \
	buster/updates main contrib non-free \
	>> /etc/apt/sources.list
RUN apt-get update
RUN apt-get build-dep -y libwebp ffmpeg
RUN mkdir /src
RUN git clone \
	--branch 1.0.3 \
	--depth 1 \
	https://chromium.googlesource.com/webm/libwebp \
	/src/libwebp
RUN git clone \
	--branch release/4.3 \
	--depth 1 \
	https://github.com/FFmpeg/FFmpeg.git \
	/src/FFmpeg
WORKDIR /src/libwebp
RUN ./autogen.sh
RUN ./configure
RUN nice -n 19 make -j $(nproc)
RUN make install
WORKDIR /src/FFmpeg
RUN ./configure
RUN nice -n 19 make -j $(nproc)
RUN make install

WORKDIR /app

# Try to cache deps
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
