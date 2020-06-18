package thumbnailer

import (
	"fmt"
	"testing"
)

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

func TestDimensionConstraints(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		name   string
		constr Dims
	}{
		{
			name: "square",
			constr: Dims{
				Width:  200,
				Height: 200,
			},
		},
		{
			name: "rect tall",
			constr: Dims{
				Width:  100,
				Height: 200,
			},
		},
		{
			name: "rect wide",
			constr: Dims{
				Width:  200,
				Height: 100,
			},
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			f := openSample(t, "non_square.png")
			defer f.Close()

			_, thumb, err := Process(f, Options{
				ThumbDims: c.constr,
			})
			if err != nil {
				t.Fatal(err)
			}

			m := thumb.Bounds().Max
			if uint(m.X) > c.constr.Width || uint(m.Y) > c.constr.Height {
				t.Fatalf(
					"thumbnail exceeds bounds: %+v not inside  %+v",
					m,
					c.constr,
				)
			}

			writeSample(
				t,
				fmt.Sprintf(
					"non_square.png_%dx%d_thumb.png",
					c.constr.Width, c.constr.Height,
				),
				thumb,
			)
		})
	}
}
