package thumbnailer

// #include "thumbnailer.h"
import "C"
import (
	"errors"
	"image"
	"io"
	"unsafe"
)

var (
	// ErrCantThumbnail denotes the input file was valid but no thumbnail could
	// be generated for it (example: audio file with no cover art).
	ErrCantThumbnail = errors.New("thumbnail can't be generated")

	// ErrGetFrame denotes an unknown failure to retrieve a video frame
	ErrGetFrame = errors.New("failed to get frame")
)

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
		err = ffError(ret)
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
	c, err := NewFFContext(rs)
	if err != nil {
		return
	}
	defer c.Close()

	src.Length = c.Duration()

	src.HasAudio, err = c.HasStream(FFAudio)
	if err != nil {
		return
	}
	src.HasVideo, err = c.HasStream(FFVideo)
	if err != nil {
		return
	}
	c.ExtractMeta(src)
	if c.HasCoverArt() {
		thumb, err = processCoverArt(c.CoverArt(), opts)
	} else {
		if src.HasVideo {
			// TODO: Detect width and height
			// TODO: Dimension verification
			thumb, err = c.Thumbnail(opts.ThumbDims)
		} else {
			err = ErrCantThumbnail
		}
	}
	return
}

// // processImage generates a thumbnail from a source image buffer. If width and
// // height are non-zero, buf is assumed to be a raw RGBA image.
// func processImage(src Source, opts Options) (Source, Thumbnail, error) {
// 	srcC := C.struct_Buffer{
// 		data:   (*C.uint8_t)(C.CBytes(src.Data)),
// 		size:   C.size_t(len(src.Data)),
// 		width:  C.ulong(src.Width),
// 		height: C.ulong(src.Height),
// 	}

// 	optsC := C.struct_Options{
// 		JPEGCompression: C.uint8_t(opts.JPEGQuality),
// 		PNGCompression: C.struct_CompressionRange{
// 			min: C.uint8_t(opts.PNGQuality.Min),
// 			max: C.uint8_t(opts.PNGQuality.Max),
// 		},
// 		maxSrcDims: C.struct_Dims{
// 			width:  C.ulong(opts.MaxSourceDims.Width),
// 			height: C.ulong(opts.MaxSourceDims.Height),
// 		},
// 		thumbDims: C.struct_Dims{
// 			width:  C.ulong(opts.ThumbDims.Width),
// 			height: C.ulong(opts.ThumbDims.Height),
// 		},
// 	}

// 	var thumb C.struct_Thumbnail
// 	errC := C.thumbnail(&srcC, &thumb, optsC)
// 	defer func() {
// 		if thumb.img.data != nil {
// 			C.free(unsafe.Pointer(thumb.img.data))
// 		}
// 	}()
// 	if errC != nil {
// 		var err error
// 		s := C.GoString(errC)
// 		C.free(unsafe.Pointer(errC))
// 		switch {
// 		case s == "too wide":
// 			err = ErrTooWide
// 		case s == "too tall":
// 			err = ErrTooTall
// 		case strings.Index(s, "Corrupt image") != -1 ||
// 			strings.Index(s, "Improper image header") != -1:
// 			err = ErrCorruptImage(s)
// 		default:
// 			err = errors.New(s)
// 		}
// 		return src, Thumbnail{}, err
// 	} else if thumb.img.data == nil {
// 		return src, Thumbnail{}, ErrThumbnailingUnknown
// 	}

// 	src.Width = uint(srcC.width)
// 	src.Height = uint(srcC.height)
// 	thumbnail := Thumbnail{
// 		IsPNG: bool(thumb.isPNG),
// 		Image: Image{
// 			Data: copyCBuffer(
// 				unsafe.Pointer(thumb.img.data),
// 				C.int(thumb.img.size),
// 			),
// 			Dims: Dims{
// 				Width:  uint(thumb.img.width),
// 				Height: uint(thumb.img.height),
// 			},
// 		},
// 	}
// 	return src, thumbnail, nil
// }
