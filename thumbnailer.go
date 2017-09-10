package thumbnailer

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Thumbnailing errors
var (
	ErrTooWide = errors.New("image too wide")
	ErrTooTall = errors.New("image too tall")
)

// Image stores an image of known dimensions
type Image struct {
	Data []byte
	Dims
}

// processImage generates a thumbnail from a source image buffer. If width and
// height are non-zero, buf is assumed to be a raw RGBA image.
func processImage(src *Source, opts Options) (thumb Thumbnail, err error) {
	thumb.IsPNG, err = getImageInfo(src)
	if err != nil {
		return
	}
	if err = handleDims(src, &thumb, opts); err != nil {
		return
	}
	thumb.Data, err = genThumb(src, &thumb, opts).Exec(src.Data)
	return
}

// Calculate and validate thumbnail dimensions
func handleDims(src *Source, thumb *Thumbnail, opts Options) error {
	// Check, if dims exceed maximum
	if src.Mime != "application/pdf" {
		maxW := opts.MaxSourceDims.Width
		maxH := opts.MaxSourceDims.Height
		if maxW != 0 && src.Width > maxW {
			return ErrTooWide
		}
		if maxH != 0 && src.Height > maxH {
			return ErrTooTall
		}
	}

	thumbW := opts.ThumbDims.Width
	thumbH := opts.ThumbDims.Height

	// Check, if image already fits thumbnail
	if src.Width <= thumbW && src.Height <= thumbH {
		thumb.Dims = src.Dims
	} else {
		var scale float64
		if src.Width >= src.Height { // Maintain aspect ratio
			scale = float64(src.Width) / float64(thumbW)
		} else {
			scale = float64(src.Height) / float64(thumbH)
		}
		thumb.Width = uint64(float64(src.Width) / scale)
		thumb.Height = uint64(float64(src.Height) / scale)
	}

	return nil
}

// Return commands, that generate the thumbnail from the source image
func genThumb(src *Source, thumb *Thumbnail, opts Options) pipeLine {
	dims := fmt.Sprintf("%dx%d", thumb.Width, thumb.Height)
	args := append(
		make([]string, 0, 16),
		"convert",
		"-size", dims, // Hint final max size to facilitate subsampling
		"-[0]",          // Only first frame, if applicable
		"+profile", "*", // Remove metadata
		"-thumbnail", dims+">", // Don't upscale
	)

	if thumb.IsPNG {
		args = append(
			args,
			// PNGs are compressed by pngquant and don't need to be compressed
			// here
			"-quality", "0",
			"png:-",
		)
	} else {
		args = append(
			args,
			"-quality", strconv.Itoa(int(opts.JPEGQuality)),
			"jpeg:-",
		)
	}

	cmd := make(pipeLine, 1, 2)
	cmd[0] = command("gm", args...)
	if thumb.IsPNG {
		cmd = append(cmd, command(
			"pngquant", "-",
			"--quality", fmt.Sprintf("0-%d", opts.PNGQuality),
		))
	}
	return cmd
}

// Detect image dimensions and transparency support
func getImageInfo(src *Source) (supportsAlpha bool, err error) {
	buf, err := execCommand(
		src.Data, "gm", "identify", "-[0]",
		"-format", "%W,%H,%A",
	)
	if err != nil {
		return
	}
	defer PutBuffer(buf)

	supportsAlpha, ok := parseImageInfo(buf.Bytes(), src)
	if !ok {
		err = fmt.Errorf("unparsable image information: %s", buf.String())
	}
	return
}

// Attempts to parse image information string. Returns, if succeeded.
func parseImageInfo(buf []byte, src *Source) (
	supportsAlpha bool, ok bool,
) {
	split := strings.Split(string(buf), ",")
	if len(split) != 3 {
		return
	}

	var err error
	src.Image.Width, err = strconv.ParseUint(split[0], 10, 64)
	if err != nil {
		return
	}
	src.Image.Height, err = strconv.ParseUint(split[1], 10, 64)
	if err != nil {
		return
	}

	// Ends with a newline
	supportsAlpha, err = strconv.ParseBool(split[2][:len(split[2])-1])
	if err != nil {
		return
	}

	ok = true
	return
}

// Return command for lossy PNG compression
func compressPNG(quality uint8) *exec.Cmd {
	return command("pngquant", "--quality", strconv.Itoa(int(quality)), "-")
}
