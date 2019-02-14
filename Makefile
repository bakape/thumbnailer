libs = $(addprefix -l,$(subst .a,,$(subst vendor_c/lib/lib,,$(wildcard vendor_c/lib/*.a))))
env = CGO_CFLAGS="-I$(CURDIR)/vendor_c/include" CGO_LDFLAGS="-L$(CURDIR)/vendor_c/lib $(libs)" PKG_CONFIG_PATH="$(CURDIR)/vendor_c/lib/pkgconfig"

WIN_ARCH=amd64

# Path to and target for the MXE cross environment for cross-compiling to
# win_amd64. Default value is the debian x86-static install path.
MXE_ROOT=$(HOME)/src/mxe/usr
MXE_TARGET=x86_64-w64-mingw32.static

clean:
	rm -f testdata/*_thumb.* *.exe

# Cross-compile from Unix into a Windows x86_64 static binary
# Depends on:
# 	mxe-x86-64-w64-mingw32.static-gcc
# 	mxe-x86-64-w64-mingw32.static-libidn
# 	mxe-x86-64-w64-mingw32.static-ffmpeg
#   mxe-x86-64-w64-mingw32.static-graphicsmagick
#
# To cross-compile for windows-x86 use:
# make cross_compile_windows WIN_ARCH=386 MXE_TARGET=i686-w64-mingw32.static
cross_tests_windows:
	CGO_ENABLED=1 GOOS=windows GOARCH=$(WIN_ARCH) \
	CC=$(MXE_ROOT)/bin/$(MXE_TARGET)-gcc \
	CXX=$(MXE_ROOT)/bin/$(MXE_TARGET)-g++ \
	PKG_CONFIG=$(MXE_ROOT)/bin/$(MXE_TARGET)-pkg-config \
	PKG_CONFIG_LIBDIR=$(MXE_ROOT)/$(MXE_TARGET)/lib/pkgconfig \
	PKG_CONFIG_PATH=$(MXE_ROOT)/$(MXE_TARGET)/lib/pkgconfig \
	go test -a -c -o test.exe --ldflags '-extldflags "-static"'
	wine ./test.exe

# vendor ffmpeg 
deps:
	mkdir -p vendor_c/src
	git clone https://github.com/FFmpeg/FFmpeg vendor_c/src/ffmpeg
	cd vendor_c/src/ffmpeg && git checkout n4.0.2
	cd vendor_c/src/ffmpeg && ./configure --prefix=../.. $(configure)
	cd vendor_c/src/ffmpeg && make
	cd vendor_c/src/ffmpeg && make install

build:
	$(env) go build