[![GoDoc](https://godoc.org/github.com/bakape/thumbnailer?status.svg)](https://godoc.org/github.com/bakape/thumbnailer)
[![Build Status](https://travis-ci.com/bakape/thumbnailer.svg?branch=master)](https://travis-ci.com/bakape/thumbnailer)
# thumbnailer
Package thumbnailer provides a more efficient image/video/audio/PDF thumbnailer
than available with native Go processing libraries through GraphicsMagic and
ffmpeg bindings.

For a comprehensive list of file formats supported by default, check the `matchers` slice in `mime.go`.

## Dependencies
* Go >= 1.10
* C11 compiler
* make
* pkg-config
* pthread
* ffmpeg >= 3.1 libraries (libavcodec, libavutil, libavformat, libswscale)

NB: ffmpeg should be compiled with all the dependency libraries for
formats you want to process. On most Linux distributions you should be fine with
the packages in the stock repositories.
