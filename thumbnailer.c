#include "thumbnailer.h"
#include <magick/pixel_cache.h>
#include <string.h>

#ifndef MIT_LICENSE
#include "compress_png.h"
#endif // MIT_LICENSE

// Iterates over all pixels and checks, if any transparency present
static char* hasTransparency(
    const Image* img, bool* need_PNG, ExceptionInfo* ex)
{
    // No alpha channel
    if (!img->matte) {
        return NULL;
    }

    // Transparent pixels are most likely to also be in the first row, so
    // retrieve one row at a time. It is also more performant to retrieve entire
    // rows instead of individual pixels.
    for (unsigned long i = 0; i < img->rows; i++) {
        const PixelPacket* packets
            = AcquireImagePixels(img, 0, i, img->columns, 1, ex);
        if (!packets) {
            return format_magick_exception(ex);
        }
        for (unsigned long j = 0; j < img->columns; j++) {
            if (packets[j].opacity > 0) {
                *need_PNG = true;
                return NULL;
            }
        }
    }

    return NULL;
}

// Convert thumbnail to appropriate file type and write to buffer
static char* writeThumb(Image* img, struct Thumbnail* thumb,
    const struct Options opts, ExceptionInfo* ex)
{
    char* err = NULL;
    char* format = NULL;
    bool need_PNG = false;
    ImageInfo* info = NULL;

    if (strcmp(img->magick, "JPEG")) {
        err = hasTransparency(img, &need_PNG, ex);
        if (err) {
            goto end;
        }
    }

// Don't need ImageInfo, if we are using lossy PNG compression
#ifndef MIT_LICENSE
    if (!need_PNG)
#endif
        info = CloneImageInfo(NULL);
    if (need_PNG) {
        thumb->isPNG = true;
#ifndef MIT_LICENSE
        return compress_png(img, thumb, opts.PNGCompression);
#else
        format = "PNG";
        info->quality = 105;
#endif // MIT_LICENSE
    } else {
        format = "JPEG";
        info->quality = get_quality(75, opts.JPEGCompression);
    }
    strcpy(info->magick, format);
    thumb->img.data = ImageToBlob(info, img, &thumb->img.size, ex);

end:
    DestroyImageInfo(info);
    return err;
}

// Swap original image with new temporary image
#define SWAP_TMP                                                               \
    if (!tmp) {                                                                \
        goto end;                                                              \
    }                                                                          \
    DestroyImage(img);                                                         \
    img = tmp;

char* thumbnail(
    struct Buffer* src, struct Thumbnail* thumb, const struct Options opts)
{
    Image *img = NULL, *tmp = NULL;
    char* err = NULL;
    ImageInfo* info = CloneImageInfo(NULL);
    ExceptionInfo ex;
    GetExceptionInfo(&ex);

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

    img = BlobToImage(info, src->data, src->size, &ex);
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
            err = copy_string("too wide");
            goto end;
        }
        if (maxH && img->rows > maxH) {
            err = copy_string("too tall");
            goto end;
        }
    }

    // Rotate image based on EXIF metadata, if needed
    if (img->orientation > TopLeftOrientation) {
        tmp = AutoOrientImage(img, img->orientation, &ex);
        SWAP_TMP;
    }

    // Strip EXIF metadata, if any. Fail silently.
    DeleteImageProfile(img, "EXIF");

    // Maintain aspect ratio
    double scale;
    if (img->columns >= img->rows) {
        scale = (double)(img->columns) / (double)(opts.thumbDims.width);
    } else {
        scale = (double)(img->rows) / (double)(opts.thumbDims.height);
    }
    thumb->img.width = (unsigned long)((double)img->columns / scale);
    thumb->img.height = (unsigned long)((double)img->rows / scale);

    // Subsample to 4 times the thumbnail size. A decent enough compromise
    // between quality and performance for images around the thumbnail size
    // and much bigger ones.
    tmp = SampleImage(img, thumb->img.width * 4, thumb->img.height * 4, &ex);
    SWAP_TMP;

    // Scale to thumbnail size
    tmp = ResizeImage(
        img, thumb->img.width, thumb->img.height, BoxFilter, 1, &ex);
    SWAP_TMP;

    err = writeThumb(img, thumb, opts, &ex);

end:
    if (img) {
        DestroyImage(img);
    }
    DestroyImageInfo(info);
    if (!err) {
        err = format_magick_exception(&ex);
    }
    DestroyExceptionInfo(&ex);
    return err;
}
