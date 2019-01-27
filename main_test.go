package thumbnailer

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

var samples = []string{
	"no_cover.mp4",
	"no_sound.mkv",
	"no_sound.ogg",
	"sample.gif",
	"with_sound.avi",
	"no_cover.flac",
	"no_cover.ogg",
	"no_sound.mov",
	"no_sound.webm",
	"sample.jpg",
	"sample.tiff",
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
	"sample.bmp",
	"sample.png",
	"with_cover.flac",
	"with_sound.mp4",
	"odd_dimensions.webm", // Unconventional dims for a YUV stream
	"alpha.webm",
	"start_black.webm", // Check the histogram thumbnailing
	"exif_orientation.jpg",
	"rare_brand.mp4",
}

func TestProcess(t *testing.T) {
	t.Parallel()

	opts := Options{
		ThumbDims: Dims{150, 150},
	}

	for i := range samples {
		sample := samples[i]
		t.Run(sample, func(t *testing.T) {
			// t.Parallel()

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
			t.Logf("thumb: %v\n", thumb)
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

func TestDimensionValidation(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		name, file string
		maxW, maxH uint
		err        error
	}{
		{
			name: "width check disabled",
			file: "too wide.jpg",
		},
		{
			name: "too wide",
			file: "too wide.jpg",
			maxW: 2000,
			err:  ErrTooWide,
		},
		{
			name: "height check disabled",
			file: "too tall.jpg",
		},
		{
			name: "too tall",
			file: "too tall.jpg",
			maxH: 2000,
			err:  ErrTooTall,
		},
		{
			name: "pdf pass through",
			file: "sample.pdf",
			maxH: 1,
			maxW: 1,
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			opts := Options{
				ThumbDims: Dims{
					Width:  150,
					Height: 150,
				},
				MaxSourceDims: Dims{
					Width:  c.maxW,
					Height: c.maxH,
				},
			}

			f := openSample(t, c.file)
			defer f.Close()

			_, _, err := Process(f, opts)
			if err != c.err {
				t.Fatalf("unexpected error: `%s` : `%s`", c.err, err)
			}
		})
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

func TestMetadataExtraction(t *testing.T) {
	t.Parallel()

	f := openSample(t, "title.mp3")
	defer f.Close()

	src, _, err := Process(f, Options{})
	if err != nil && err != ErrCantThumbnail {
		t.Fatal(err)
	}
	if src.Artist != "Test Artist" {
		t.Errorf("unexpected artist: Test Artist : %s", src.Artist)
	}
	if src.Title != "Test Title" {
		t.Errorf("unexpected title: Test Title: %s", src.Title)
	}
}

func TestWebmAlpha(t *testing.T) {
	t.Parallel()

	f := openSample(t, "alpha.webm")
	defer f.Close()

	_, _, err := Process(f, Options{
		ThumbDims: Dims{
			Width:  150,
			Height: 150,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Called on `go test -args all`
func TestPanic(t *testing.T) {
	type B struct {
		c int
	}
	type A struct {
		b *B
	}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	a := A{nil}
	fmt.Println(a.b.c)
}
