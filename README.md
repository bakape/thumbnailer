[![GoDoc](https://godoc.org/github.com/bakape/go-thumbnailer?status.svg)](https://godoc.org/github.com/bakape/go-thumbnailer)
# go-thumbnailer
Package thumbnailer provides a more efficient image/video/audio/PDF thumbnailer
than available with native Go processing libraries through GraphicsMagic and
ffmpeg bindings.

##Dependencies
* GCC or Clang
* make
* pkg-config
* pthread
* ffmpeg >= 3.0 libraries (libavcodec, libavutil, libavformat) compiled with:
    * libvpx
    * libvorbis
    * libopus
    * libtheora
    * libx264
    * libmp3lame
* GraphicsMagick compiler with:
    * zlib
    * libpng
    * libjpeg
    * postscript
