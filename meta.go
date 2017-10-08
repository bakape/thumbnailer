package thumbnailer

// #include "meta.h"
import "C"
import "unsafe"

// ExtractMeta retrieves title and artist for source, if present
func (c *FFContext) ExtractMeta(src *Source) {
	meta := C.retrieve_meta(c.avFormatCtx)
	if meta.title != nil {
		src.Title = C.GoString(meta.title)
		C.free(unsafe.Pointer(meta.title))
	}
	if meta.artist != nil {
		src.Artist = C.GoString(meta.artist)
		C.free(unsafe.Pointer(meta.artist))
	}
}
