#pragma once
#include "util.h"
#include <magick/api.h>
#include <stdbool.h>

struct Thumbnail {
	bool isPNG;
	struct Buffer img;
};

struct Options {
	uint8_t JPEGCompression;
	struct Dims maxSrcDims, thumbDims;
};

int thumbnail(struct Buffer *src,
			  struct Thumbnail *thumb,
			  const struct Options opts,
			  ExceptionInfo *ex);
static int writeThumb(Image *img,
					  struct Thumbnail *thumb,
					  const struct Options opts,
					  ExceptionInfo *ex);
static int
hasTransparency(const Image const *img, bool *needPNG, ExceptionInfo *ex);
