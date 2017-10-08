#include "thumbnailer.h"
#include <magick/pixel_cache.h>
#include <string.h>

int thumbnail(struct Buffer* src, struct Thumbnail* thumb,
    const struct Options opts, ExceptionInfo* ex)
{
    ImageInfo* info = NULL;
    Image *img = NULL, *sampled = NULL, *scaled = NULL;
    double scale;
    int err = 0;

    // Read image
    info = CloneImageInfo(NULL);
    GetExceptionInfo(ex);

    // Read only the first frame/page of GIFs and PDFs
    info->subimage = 0;
    info->subrange = 1;

    // If width and height are already defined, then a frame from ffmpeg has
    // been passed
    if (src->width && src->height) {
        strcpy(info->magick, "RGBA");
        char* buf = malloc(128);
        int over = snprintf(buf, 128, "%lux%lu", src->width, src->height);
        if (over > 0) {
            buf = realloc(buf, 128 + (size_t)over);
            sprintf(buf, "%lux%lu", src->width, src->height);
        }
        info->size = buf;
        info->depth = 8;
    }

    img = BlobToImage(info, src->data, src->size, ex);
    if (!img) {
        goto end;
    }
    src->width = img->columns;
    src->height = img->rows;

    // Validate dimensions
    if (strcmp(img->magick, "PDF")) {
        const unsigned long maxW = opts.maxSrcDims.width;
        const unsigned long maxH = opts.maxSrcDims.height;
        if (maxW && img->columns > maxW) {
            err = 2;
            goto end;
        }
        if (maxH && img->rows > maxH) {
            err = 3;
            goto end;
        }
    }

    const unsigned long thumbW = opts.thumbDims.width;
    const unsigned long thumbH = opts.thumbDims.height;

    // Image already fits thumbnail
    if (img->columns <= thumbW && img->rows <= thumbH) {
        thumb->img.width = img->columns;
        thumb->img.height = img->rows;
        err = writeThumb(img, thumb, opts, ex);
        goto end;
    }

    // Maintain aspect ratio
    if (img->columns >= img->rows) {
        scale = (double)(img->columns) / (double)(thumbW);
    } else {
        scale = (double)(img->rows) / (double)(thumbH);
    }
    thumb->img.width = (unsigned long)((double)img->columns / scale);
    thumb->img.height = (unsigned long)((double)img->rows / scale);

    // Subsample to 4 times the thumbnail size. A decent enough compromise
    // between quality and performance for images arround the thumbnail size
    // and much bigger ones.
    sampled = SampleImage(img, thumb->img.width * 4, thumb->img.height * 4, ex);
    if (!sampled) {
        goto end;
    }

    // Scale to thumbnail size
    scaled = ResizeImage(
        sampled, thumb->img.width, thumb->img.height, BoxFilter, 1, ex);
    if (!scaled) {
        goto end;
    }

    err = writeThumb(scaled, thumb, opts, ex);

end:
    if (img) {
        DestroyImage(img);
    }
    if (info) {
        DestroyImageInfo(info);
    }
    if (sampled) {
        DestroyImage(sampled);
    }
    if (scaled) {
        DestroyImage(scaled);
    }
    if (!err) {
        return thumb->img.data == NULL;
    }
    return err;
}

// Convert thumbnail to apropriate file type and write to buffer
static int writeThumb(Image* img, struct Thumbnail* thumb,
    const struct Options opts, ExceptionInfo* ex)
{
    ImageInfo* info = CloneImageInfo(NULL);
    char* format = NULL;
    bool needPNG = false;

    if (strcmp(img->magick, "JPEG")) {
        int err = hasTransparency(img, &needPNG, ex);
        if (err) {
            DestroyImageInfo(info);
            return err;
        }
    }
    if (needPNG) {
        format = "PNG";
        info->quality = 105;
        thumb->isPNG = true;
    } else {
        format = "JPEG";
        info->quality = opts.JPEGCompression;
    }
    strcpy(info->magick, format);
    strcpy(img->magick, format);
    thumb->img.data = ImageToBlob(info, img, &thumb->img.size, ex);

    DestroyImageInfo(info);
    return 0;
}

// Itterates over all pixels and checks, if any transparency present
static int hasTransparency(
    const Image const* img, bool* needPNG, ExceptionInfo* ex)
{
    // No alpha channel
    if (!img->matte) {
        return 0;
    }

    // Transparent pixels are most likely to also be in the first row, so
    // retrieve one row at a time. It is also more performant to retrieve entire
    // rows instead of individual pixels.
    for (unsigned long i = 0; i < img->rows; i++) {
        const PixelPacket* packets
            = AcquireImagePixels(img, 0, i, img->columns, 1, ex);
        if (!packets) {
            return 1;
        }
        for (unsigned long j = 0; j < img->columns; j++) {
            if (packets[j].opacity > 0) {
                *needPNG = true;
                return 0;
            }
        }
    }

    return 0;
}
