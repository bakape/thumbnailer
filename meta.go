package thumbnailer

// #include "meta.h"
import "C"

// Meta retrieves title and artist for source, if present
func (c *FFContext) Meta() (m Meta) {
	meta := C.retrieve_meta(c.avFormatCtx)
	if meta.title != nil {
		m.Title = C.GoString(meta.title)
		sanitize(&m.Title)
	}
	if meta.artist != nil {
		m.Artist = C.GoString(meta.artist)
		sanitize(&m.Artist)
	}
	return
}
