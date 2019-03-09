FROM debian:testing
COPY . .
RUN apt-get update
RUN apt-get dist-upgrade -y
RUN apt-get install -y build-essential pkg-config libpth-dev libavcodec-dev libavutil-dev libavformat-dev libswscale-dev golang git
