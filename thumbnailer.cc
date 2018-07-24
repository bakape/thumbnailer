extern "C" {
#include "thumbnailer.h"
#include <string.h>
}
#include "util.hh"
#include <Magick++.h>
#include <functional>
#include <stdexcept>

#ifndef MIT_LICENSE
#include "compress_png.hh"
#endif // MIT_LICENSE

// Iterates over all pixels and checks, if any transparency present
static bool has_transparency(const Magick::Image& img)
{
    // No alpha channel
    if (!img.matte()) {
        return false;
    }

    // Transparent pixels are most likely to also be in the first row, so
    // retrieve one row at a time. It is also more performant to retrieve
    // entire rows instead of individual pixels.
    for (unsigned long i = 0; i < img.rows(); i++) {
        const auto packets = img.getConstPixels(0, i, img.columns(), 1);
        for (unsigned long j = 0; j < img.columns(); j++) {
            if (packets[j].opacity > 0) {
                return true;
            }
        }
    }

    return false;
}

// Convert thumbnail to appropriate file type and write to buffer
static void write_thumb(
    Magick::Image& img, struct Thumbnail* thumb, const struct Options opts)
{
    const bool need_png = img.magick() != "JPEG" && has_transparency(img);
    if (need_png) {
        thumb->isPNG = true;
#ifndef MIT_LICENSE
        return compress_png(img, thumb, opts.PNGCompression);
#else
        img.magick("PNG");
        img.quality(105);
        img.depth(8);
#endif // MIT_LICENSE
    } else {
        img.magick("JPEG");
        img.quality(get_quality(75, opts.JPEGCompression));
    }

    Magick::Blob out;
    img.write(&out);
    thumb->img.data = (uint8_t*)malloc(out.length());
    memcpy(thumb->img.data, out.data(), out.length());
    thumb->img.size = out.length();
}

static void _thumbnail(
    struct Buffer* src, struct Thumbnail* thumb, const struct Options opts)
{
    Magick::Blob blob;
    blob.updateNoCopy(src->data, src->size, Magick::Blob::MallocAllocator);

    // If width and height are already defined, then a frame from ffmpeg has
    // been passed
    Magick::Image img = (src->width && src->height)
        ? Magick::Image(
              blob, Magick::Geometry(src->width, src->height), 8, "RGBA")
        : Magick::Image(blob, Magick::Geometry(src->width, src->height));
    src->width = img.columns();
    src->height = img.rows();

    // Validate dimensions
    if (img.magick() != "PDF") {
        const unsigned long maxW = opts.maxSrcDims.width;
        const unsigned long maxH = opts.maxSrcDims.height;
        if (maxW && img.columns() > maxW) {
            throw std::invalid_argument("too wide");
        }
        if (maxH && img.rows() > maxH) {
            throw std::invalid_argument("too tall");
        }
    }

    // Rotate image based on EXIF metadata, if needed
    if (img.orientation() > Magick::OrientationType::TopLeftOrientation) {
        // As of writing the Magick::Image::autoOrient() method is not yet in
        // the GraphicsMagick++ version in Debian stable repos. Inlined it here.
        MagickLib::ExceptionInfo exceptionInfo;
        MagickLib::GetExceptionInfo(&exceptionInfo);
        MagickLib::Image* newImage
            = AutoOrientImage(img.image(), img.orientation(), &exceptionInfo);
        img.replaceImage(newImage);
        Magick::throwException(exceptionInfo);
    }
    // Strip EXIF metadata, if any
    img.strip();

    const unsigned long thumbW = opts.thumbDims.width;
    const unsigned long thumbH = opts.thumbDims.height;
    if (img.columns() <= thumbW && img.rows() <= thumbH) {
        // Image already fits thumbnail
        thumb->img.width = img.columns();
        thumb->img.height = img.rows();
    } else {
        // Maintain aspect ratio
        const double scale = img.columns() >= img.rows()
            ? (double)img.columns() / (double)thumbW
            : (double)img.rows() / (double)thumbH;
        thumb->img.width = (unsigned long)((double)img.columns() / scale);
        thumb->img.height = (unsigned long)((double)img.rows() / scale);

        img.thumbnail(Magick::Geometry(thumb->img.width, thumb->img.height));
    }

    write_thumb(img, thumb, opts);
}

extern "C" char* thumbnail(
    struct Buffer* src, struct Thumbnail* thumb, const struct Options opts)
{
    return pass_exception([=]() -> char* {
        _thumbnail(src, thumb, opts);
        return NULL;
    });
}
