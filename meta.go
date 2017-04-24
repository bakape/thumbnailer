package thumbnailer

// #include "meta.h"
import "C"

// ExtractMeta retrieves title and artist for source, if present
func (c *FFContext) ExtractMeta(src *Source) {
	meta := C.retrieve_meta(c.avFormatCtx)
	if meta.title != nil {
		title := C.GoString(meta.title)
		src.Title = &title
	}

	if meta.artist != nil {
		artist := C.GoString(meta.artist)
		src.Artist = &artist
	}
}
