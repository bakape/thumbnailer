package thumbnailer

import (
	"errors"
	"fmt"
)

// Alias for backwards compatibility
type UnsupportedMIMEError = ErrUnsupportedMIME

// Indicates the MIME type of the file could not be detected as a supported type
// or was not in the AcceptedMimeTypes list, if defined.
type ErrUnsupportedMIME string

func (e ErrUnsupportedMIME) Error() string {
	return fmt.Sprintf("unsupported MIME type: %s", string(e))
}

// Thumbnailing errors
var (
	ErrTooWide             = ErrInvalidImage("image too wide")
	ErrTooTall             = ErrInvalidImage("image too tall")
	ErrThumbnailingUnknown = errors.New("unknown thumbnailing error")
)

// Indicates and invalid image has been passed for processing
type ErrInvalidImage string

func (e ErrInvalidImage) Error() string {
	return fmt.Sprintf("invalid image: %s", string(e))
}

// Source image is corrupt and could not be decoded
type ErrCorruptImage string

func (e ErrCorruptImage) Error() string {
	return string(e)
}
