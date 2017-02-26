[![GoDoc](https://godoc.org/github.com/bakape/thumbnailer?status.svg)](https://godoc.org/github.com/bakape/thumbnailer)
# thumbnailer
Package thumbnailer provides a more efficient image/video/audio/PDF thumbnailer
than available with native Go processing libraries through GraphicsMagic and
ffmpeg bindings.


For a comprehensive list of file formats supported by default, check the `matchers` slice in `mime.go`.

##Dependencies
* GCC or Clang
* make
* pkg-config
* pthread
* ffmpeg >= 3.0 libraries (libavcodec, libavutil, libavformat):
* GraphicsMagick

NB: ffmpeg and GM should be compiled with all the dependency libraries for
formats you want to process. On most Linux distributions you should be fine with
the packages in the stock repositories.
