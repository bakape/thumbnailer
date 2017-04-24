package thumbnailer

// #include "meta.h"
import "C"

// ExtractMeta retrieves title and artist for source, if present
func (c *FFContext) ExtractMeta(src *Source) {
	meta := C.retrieve_meta(c.avFormatCtx)
	if meta.title != nil {
		src.Title = C.GoString(meta.title)
	}

	if meta.artist != nil {
		src.Artist = C.GoString(meta.artist)
	}
}
