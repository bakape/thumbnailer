package thumbnailer

// #cgo pkg-config: libavcodec libavutil libavformat libswscale
// #cgo CFLAGS: -std=c11
// #include "video.h"
import "C"
import (
	"bytes"
	"errors"
	"unsafe"
)

var (
	// ErrNoStreams denotes no decodeable audio or video streams were found in
	// a media container
	ErrNoStreams = errors.New("no decodeable video or audio streams found")

	// ErrGetFrame denotes an unknown failure to retrieve a video frame
	ErrGetFrame = errors.New("failed to get frame")
)

// Thumbnail extracts the first frame of the video
func (c *FFContext) Thumbnail() (thumb Image, err error) {
	ci, err := c.codecContext(FFVideo)
	if err != nil {
		return
	}

	var img C.struct_Buffer
	ret := C.extract_video_image(&img, c.avFormatCtx, ci.ctx, ci.stream)
	switch {
	case ret != 0:
		err = ffError(ret)
	case img.data == nil:
		err = ErrGetFrame
	default:
		p := unsafe.Pointer(img.data)
		thumb.Data = copyCBuffer(p, C.int(img.size))
		C.free(p)
		thumb.Width = uint(img.width)
		thumb.Height = uint(img.height)
	}
	return
}

func processVideo(source Source, opts Options) (
	src Source, thumb Thumbnail, err error,
) {
	src = source

	c, err := NewFFContext(bytes.NewReader(src.Data))
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
	if !src.HasVideo {
		// As of writing ffmpeg does not support cover art in neither MP4-like
		// containers or OGG, so consider these unthumbnailable
		if !src.HasAudio {
			err = ErrNoStreams
		}
		return
	}

	c.ExtractMeta(&src)

	original := src.Data
	src.Image, err = c.Thumbnail()
	if err != nil {
		return
	}
	src, thumb, err = processImage(src, opts)
	ReturnBuffer(src.Data)
	src.Data = original
	return
}
