// +build !MIT

package thumbnailer

// #cgo CXXFLAGS: -I${SRCDIR}/libimagequant -I${SRCDIR}/lodepng
// #cgo CFLAGS: -I${SRCDIR}/libimagequant
// #cgo LDFLAGS: -lm
// #include "blur.c"
// #include "kmeans.c"
// #include "libimagequant.c"
// #include "mediancut.c"
// #include "mempool.c"
// #include "nearest.c"
// #include "pam.c"
import "C"
