package thumbnailer

// #include "ffmpeg.h"
import "C"
import (
	"errors"
	"fmt"
	"io"
)

// Thumbnailing errors
var (
	ErrTooWide             = ErrInvalidImage("image too wide")
	ErrTooTall             = ErrInvalidImage("image too tall")
	ErrThumbnailingUnknown = errors.New("unknown thumbnailing error")

	// ErrCantThumbnail denotes the input file was valid but no thumbnail could
	// be generated for it (example: audio file with no cover art).
	ErrCantThumbnail = errors.New("thumbnail can't be generated")

	// ErrGetFrame denotes an unknown failure to retrieve a video frame
	ErrGetFrame = errors.New("failed to get frame")

	// ErrStreamNotFound denotes no steam of this media type was found
	ErrStreamNotFound = errors.New("no stream of this type found")
)

// Indicates the MIME type of the file could not be detected as a supported type
// or was not in the AcceptedMimeTypes list, if defined.
type ErrUnsupportedMIME string

func (e ErrUnsupportedMIME) Error() string {
	return fmt.Sprintf("unsupported MIME type: %s", string(e))
}

// Indicates and invalid image has been passed for processing
type ErrInvalidImage string

func (e ErrInvalidImage) Error() string {
	return fmt.Sprintf("invalid image: %s", string(e))
}

// ErrorCovert wraps an error that happened during cover art thumbnailing
type ErrCoverArt struct {
	Err error
}

func (e ErrCoverArt) Error() string {
	return "cover art: " + e.Err.Error()
}

// ErrArchive wraps an error that happened during thumbnailing a file in zip
// archive
type ErrArchive struct {
	Err error
}

func (e ErrArchive) Error() string {
	return "archive: " + e.Err.Error()
}

// Cast FFmpeg error to Go error
func castError(err C.int) error {
	switch err {
	case C.AVERROR_EOF:
		return io.EOF
	case C.AVERROR_STREAM_NOT_FOUND:
		return ErrStreamNotFound
	default:
		return AVError(err)
	}
}

// AVError converts an FFmpeg error code to a Go error with a human-readable
// error message
type AVError C.int

// Error formats the FFmpeg error in human-readable format
func (f AVError) Error() string {
	buf := C.malloc(1024)
	defer C.free(buf)
	C.av_strerror(C.int(f), (*C.char)(buf), 1024)
	return fmt.Sprintf("ffmpeg: %s", C.GoString((*C.char)(buf)))
}

// Code returns the underlying AVERROR error code
func (f AVError) Code() C.int {
	return C.int(f)
}
