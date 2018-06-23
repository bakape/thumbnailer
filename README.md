[![GoDoc](https://godoc.org/github.com/bakape/thumbnailer?status.svg)](https://godoc.org/github.com/bakape/thumbnailer)
# thumbnailer
Package thumbnailer provides a more efficient image/video/audio/PDF thumbnailer
than available with native Go processing libraries through GraphicsMagic and
ffmpeg bindings.

For a comprehensive list of file formats supported by default, check the `matchers` slice in `mime.go`.

## License
GNU GENERAL PUBLIC LICENSE / MIT License

Depending on how the project is built it can be licensed under either MIT or
GPLv3. Thumbnailer links against the GPLv3-licensed libimagequant for lossy PNG
thumbnail compression by default and thus also becomes applicable under the
GPLv3. To build thumbnailer without this feature under the MIT license, please
specify `--tags=MIT` when building the project. See LICENSE for more details.

## Dependencies
* Go >= 1.10
* C11 and C++17 compilers
* make
* pkg-config
* pthread
* ffmpeg >= 3.1 libraries (libavcodec, libavutil, libavformat, libswscale)
* GraphicsMagick++

NB: ffmpeg and GM should be compiled with all the dependency libraries for
formats you want to process. On most Linux distributions you should be fine with
the packages in the stock repositories.
