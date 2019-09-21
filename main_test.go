package thumbnailer

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

var samples = [...]string{
	"no_cover.mp4",
	"no_sound.mkv",
	"no_sound.ogg",
	"sample.gif",
	"with_sound.avi",
	"no_cover.flac",
	"no_cover.ogg",
	"no_sound.mov",
	"no_sound.webm",
	"with_sound_vp9.webm",
	"sample.jpg",
	"with_cover.mp3",
	"with_sound.mkv",
	"with_sound.ogg",
	"no_sound.avi",
	"no_sound.mp4",
	"no_sound.wmv",
	"sample.webp",
	"with_sound.mov",
	"with_sound.webm",
	"no_cover.mp3",
	"no_magic.mp3", // No magic numbers
	"no_sound.flv",
	"sample.png",
	"with_cover.flac",
	"with_sound.mp4",
	"odd_dimensions.webm", // Unconventional dims for a YUV stream
	"alpha.webm",
	"start_black.webm", // Check the histogram thumbnailing
	"exif_orientation.jpg",
	"rare_brand.mp4",
	"invalid_data.jpg", // Check handling images with some invalid data
	"sample.zip",
	"sample.rar",
	"too small.png",
	"exact_thumb_size.jpg",
	"meta_segfault.mp4",
}

var ignore = map[string]bool{
	"invalid_data.jpg": true,
	"sample.zip":       true,
	"sample.rar":       true,
}

func TestProcess(t *testing.T) {
	t.Parallel()

	opts := Options{
		ThumbDims: Dims{150, 150},
	}

	for i := range samples {
		sample := samples[i]
		t.Run(sample, func(t *testing.T) {
			t.Parallel()

			f := openSample(t, sample)
			defer f.Close()

			src, thumb, err := Process(f, opts)
			if err != nil && err != ErrCantThumbnail {
				t.Fatal(err)
			}

			if err != ErrCantThumbnail {
				name := fmt.Sprintf(`%s_thumb.png`, sample)
				writeSample(t, name, thumb)
			}

			t.Logf("src:   %v\n", src)
			if thumb != nil {
				t.Logf("thumb: %v\t\n", thumb.Bounds())
			}
		})
	}
}

func openSample(t *testing.T, name string) *os.File {
	t.Helper()

	f, err := os.Open(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func writeSample(t *testing.T, name string, img image.Image) {
	t.Helper()

	f, err := os.Create(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	png.Encode(f, img)
	if err != nil {
		t.Fatal(err)
	}
}

func TestErrorPassing(t *testing.T) {
	t.Parallel()

	f := openSample(t, "sample.txt")
	defer f.Close()

	_, _, err := Process(f, Options{
		ThumbDims: Dims{
			Width:  150,
			Height: 150,
		},
	})
	if err == nil {
		t.Fatal(`expected error`)
	}
}

func TestSourceAlreadyThumbSize(t *testing.T) {
	t.Parallel()

	f := openSample(t, "too small.png")
	defer f.Close()

	_, thumb, err := Process(f, Options{
		ThumbDims: Dims{
			Width:  150,
			Height: 150,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	dims := thumb.Bounds().Max
	if dims.X != 121 {
		t.Errorf("unexpected width: 121 : %d", dims.X)
	}
	if dims.Y != 150 {
		t.Errorf("unexpected height: 150: %d", dims.Y)
	}
}
