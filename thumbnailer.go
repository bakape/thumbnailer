package thumbnailer

// #include "thumbnailer.h"
// #include <libavutil/log.h>
import "C"
import (
	"image"
	"io"
	"unsafe"
)

func init() {
	C.av_log_set_level(C.AV_LOG_ERROR)
}

// Thumbnail generates a thumbnail from a representative frame of the media.
// Images count as one frame media.
func (c *FFContext) Thumbnail(dims Dims) (thumb image.Image, err error) {
	ci, err := c.codecContext(FFVideo)
	if err != nil {
		return
	}

	var img C.struct_Buffer
	defer func() {
		if img.data != nil {
			C.free(unsafe.Pointer(img.data))
		}
	}()
	ret := C.generate_thumbnail(&img, c.avFormatCtx, ci.ctx, ci.stream,
		C.struct_Dims{
			width:  C.ulong(dims.Width),
			height: C.ulong(dims.Height),
		})
	switch {
	case ret != 0:
		err = castError(ret)
	case img.data == nil:
		err = ErrGetFrame
	default:
		thumb = &image.RGBA{
			Pix:    copyCBuffer(img),
			Stride: 4 * int(img.width),
			Rect: image.Rectangle{
				Max: image.Point{
					X: int(img.width),
					Y: int(img.height),
				},
			},
		}
	}
	return
}

func processMedia(rs io.ReadSeeker, src *Source, opts Options,
) (
	thumb image.Image, err error,
) {
	c, err := newFFContextWithFormat(rs, inputFormats[src.Mime])
	if err != nil {
		return
	}
	defer c.Close()

	// TODO: EXIF orientation

	src.Length = c.Length()
	src.Meta = c.Meta()
	src.HasAudio, err = c.HasStream(FFAudio)
	if err != nil {
		return
	}
	src.HasVideo, err = c.HasStream(FFVideo)
	if err != nil {
		return
	}
	if src.HasVideo {
		src.Dims, err = c.Dims()
		if err != nil {
			return
		}
	}

	if c.HasCoverArt() {
		thumb, err = processCoverArt(c.CoverArt(), opts)
		switch err {
		case nil:
			return
		case ErrCantThumbnail:
			// Try again on the container itself, if cover art thumbnailing
			// fails
		default:
			return
		}
	}

	if src.HasVideo {
		max := opts.MaxSourceDims
		if max.Width != 0 && src.Width > max.Width {
			err = ErrTooWide
			return
		}
		if max.Height != 0 && src.Height > max.Height {
			err = ErrTooTall
			return
		}

		thumb, err = c.Thumbnail(opts.ThumbDims)
	} else {
		err = ErrCantThumbnail
	}
	return
}
