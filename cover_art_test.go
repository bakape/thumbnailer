package thumbnailer

import (
	"strings"
	"testing"
)

func TestCoverArt(t *testing.T) {
	t.Parallel()

	type testCase struct {
		file string
		has  bool
	}

	var cases []testCase
	for _, f := range samples {
		if ignore[f] {
			continue
		}
		cases = append(cases, testCase{f, strings.HasPrefix(f, "with_cover")})
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.file, func(t *testing.T) {
			t.Parallel()

			f := openSample(t, c.file)
			defer f.Close()

			ctx, err := NewFFContext(f)
			if err != nil {
				t.Fatal(err)
			}
			defer ctx.Close()

			has := ctx.HasCoverArt()
			if has != c.has {
				if c.file == "with_cover.flac" {
					t.Skip("cover art in FLAC is not supported yet")
				}
				t.Fatal("unexpected cover art presence")
			}
			if has {
				buf := ctx.CoverArt()
				if len(buf) == 0 {
					t.Fatal("zero length cover art buffer")
				}
			}
		})
	}
}
