// +build !MIT

package thumbnailer

// #cgo CFLAGS: -I${SRCDIR}/libimagequant -I${SRCDIR}/lodepng
// #cgo LDFLAGS: -lm
// #include "blur.c"
// #include "kmeans.c"
// #include "libimagequant.c"
// #include "mediancut.c"
// #include "mempool.c"
// #include "nearest.c"
// #include "pam.c"
// #include "lodepng.cpp"
import "C"
