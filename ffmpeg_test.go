package thumbnailer

import (
	"strings"
	"testing"
)

func TestDims(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		dims Dims
	}

	var cases []testCase
	var c testCase
	for _, f := range samples {
		if ignore[f] {
			continue
		}
		c.name = f
		switch {
		case f == "with_cover.mp3":
			c.dims = Dims{1280, 720}
		case f == "sample.gif":
			c.dims = Dims{584, 720}
		case strings.HasPrefix(f, "no_cover"),
			strings.HasPrefix(f, "with_cover"):
			c.dims = Dims{0, 0}
		case strings.HasPrefix(f, "sample"):
			c.dims = Dims{1280, 720}
		default:
			continue
		}
		cases = append(cases, c)
	}

	opts := Options{
		ThumbDims: Dims{150, 150},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			f := openSample(t, c.name)
			defer f.Close()

			src, _, err := Process(f, opts)
			switch err {
			case nil:
			case ErrCantThumbnail:
			default:
				t.Fatal(err)
			}
			if src.Dims != c.dims {
				t.Fatalf("%v != %v", src.Dims, c.dims)
			}
		})
	}
}
