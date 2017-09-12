package thumbnailer

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestProcess(t *testing.T) {
	var samples = map[string]error{
		"no_cover.mp4":        ErrNoThumb,
		"no_cover.flac":       ErrNoThumb,
		"no_cover.ogg":        ErrNoThumb,
		"no_cover.mp3":        ErrNoThumb,
		"no_sound.mkv":        nil,
		"no_sound.ogg":        nil,
		"sample.gif":          nil,
		"sample.psd":          nil,
		"with_sound.avi":      nil,
		"no_sound.mov":        nil,
		"no_sound.webm":       nil,
		"sample.jpg":          nil,
		"sample.tiff":         nil,
		"with_cover.mp3":      nil,
		"with_sound.mkv":      nil,
		"with_sound.ogg":      nil,
		"no_sound.avi":        nil,
		"no_sound.mp4":        nil,
		"no_sound.wmv":        nil,
		"sample.pdf":          nil,
		"sample.webp":         nil,
		"with_sound.mov":      nil,
		"with_sound.webm":     nil,
		"no_magic.mp3":        nil, // No magic numbers
		"no_sound.flv":        nil,
		"sample.bmp":          nil,
		"sample.png":          nil,
		"with_sound.mp4":      nil,
		"odd_dimensions.webm": nil, // Unconventional dims for a YUV stream
	}

	t.Parallel()

	opts := Options{
		JPEGQuality: 90,
		ThumbDims:   Dims{150, 150},
	}

	for s, exErr := range samples {
		// Persist through other goroutines
		sample := s
		expectedErr := exErr

		t.Run(sample, func(t *testing.T) {
			t.Parallel()

			f := openSample(t, sample)
			defer f.Close()

			src, thumb, err := Process(f, opts)
			if err != expectedErr {
				t.Fatal(err)
			}

			var ext string
			if thumb.IsPNG {
				ext = "png"
			} else {
				ext = "jpg"
			}
			name := fmt.Sprintf(`%s_thumb.%s`, sample, ext)
			writeSample(t, name, thumb.Data)

			src.Data = nil
			thumb.Data = nil
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

func writeSample(t *testing.T, name string, buf []byte) {
	t.Helper()

	path := filepath.Join("testdata", name)

	// Remove previous file, if any
	_, err := os.Stat(path)
	switch {
	case os.IsExist(err):
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
	case os.IsNotExist(err):
	case err == nil:
	default:
		t.Fatal(err)
	}

	err = ioutil.WriteFile(path, buf, 0600)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDimensionValidation(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		name, file string
		maxW, maxH uint64
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
				JPEGQuality: 90,
			}

			f := openSample(t, c.file)
			defer f.Close()

			if _, _, err := Process(f, opts); err != c.err {
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
	if thumb.Width != 121 {
		t.Errorf("unexpected width: 121 : %d", thumb.Width)
	}
	if thumb.Height != 150 {
		t.Errorf("unexpected height: 150: %d", thumb.Height)
	}
}

// func TestMetadataExtraction(t *testing.T) {
// 	t.Parallel()

// 	f := openSample(t, "title.mp3")
// 	defer f.Close()

// 	src, _, err := Process(f, Options{})
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	if src.Artist != "Test Artist" {
// 		t.Errorf("unexpected artist: Test Artist : %s", src.Artist)
// 	}
// 	if src.Title != "Test Title" {
// 		t.Errorf("unexpected title: Test Title: %s", src.Title)
// 	}
// }
