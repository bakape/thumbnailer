// GraphicsMagick does not support PNG8 properly, so use lodepng instead
#include "lodepng.h"
// Because you can't include C++ files in Go files
#include "lodepng.cpp"

extern "C" {
#include "libimagequant.h"
#include "thumbnailer.h"
#include <string.h>
}
#include <Magick++.h>
#include <functional>
#include <stdexcept>

// Exception of the `prefix: message` format
class PrefixedError : public std::exception {
public:
    PrefixedError(const char* prefix, const char* s)
    {
        err += prefix;
        err += ": ";
        err += s;
    }

    const char* what() const noexcept { return err.data(); }

private:
    std::string err;
};

// libimagequant exception
class LIQError : public PrefixedError {
public:
    // Convert error message to exception
    LIQError(const char* s)
        : PrefixedError("imagequant", s)
    {
    }

    // Convert libimagequant error to textual exception
    LIQError(int e)
        : PrefixedError("imagequant", LIQError::map_error(e))
    {
    }

private:
    static const char* map_error(int e)
    {
        switch (e) {
        case LIQ_QUALITY_TOO_LOW:
            return "quality too low";
        case LIQ_VALUE_OUT_OF_RANGE:
            return "value out of range";
        case LIQ_OUT_OF_MEMORY:
            return "out of memory";
        case LIQ_ABORTED:
            return "aborted";
        case LIQ_BITMAP_NOT_AVAILABLE:
            return "bitmap not available";
        case LIQ_BUFFER_TOO_SMALL:
            return "buffer too small";
        case LIQ_INVALID_POINTER:
            return "invalid pointer";
        case LIQ_UNSUPPORTED:
            return "unsupported";
        default:
            return "unknown error";
        }
    }
};

// loadpng exception
class LoadpngError : public PrefixedError {
public:
    // Convert error message to exception
    LoadpngError(int e)
        : PrefixedError("loadpng", lodepng_error_text(e))
    {
    }
};

// Creates RAII wrapper for C type T with simple destructor D.
template <class T, void (*D)(T*)> class RAII {
public:
    RAII(T* ptr = nullptr)
        : ptr(ptr)
    {
    }

    ~RAII()
    {
        if (ptr) {
            D(ptr);
        }
    }

    // Implicit conversion to internal pointer
    operator T*() { return ptr; }

    // Override & for taking a pointer of the internal pointer
    T** operator&() { return &ptr; }

private:
    T* ptr = nullptr;
};

// Lossy PNG compression. img is reused and can be set to NULL after call in
// case of error.
static void compress_png(struct Thumbnail* thumb, double gamma,
    const struct CompressionRange quality)
{
    // Prepare handle
    RAII<liq_attr, liq_attr_destroy> handle(liq_attr_create());
    if (!handle) {
        throw LIQError("can not run on this machine");
    }
    if (quality.min <= 100 && quality.max <= 100) {
        liq_set_quality(handle, quality.min, quality.max);
    }

    // Read image into libimagequant
    RAII<void, free> raw_in;
    unsigned int _w, _h; // Dummies - we already know these
    int err = lodepng_decode32(
        (unsigned char**)&raw_in, &_w, &_h, thumb->img.data, thumb->img.size);
    if (err) {
        throw LoadpngError(err);
    }
    RAII<liq_image, liq_image_destroy> input_image(liq_image_create_rgba(
        handle, raw_in, thumb->img.width, thumb->img.height, gamma));
    if (!input_image) {
        throw LIQError("error allocating image");
    }

    // Quantize image
    RAII<liq_result, liq_result_destroy> res;
    err = liq_image_quantize(input_image, handle, &res);
    if (err) {
        throw LIQError(err);
    }

    // Write image to RGBA buffer
    err = liq_set_dithering_level(res, 1.0);
    if (err) {
        throw LIQError(err);
    }
    const size_t pixels_size = thumb->img.height * thumb->img.width;
    RAII<void, free> raw_out(malloc(pixels_size));
    err = liq_write_remapped_image(res, input_image, raw_out, pixels_size);
    if (err) {
        throw LIQError(err);
    }
    const liq_palette* palette = liq_get_palette(res);
    if (!palette) {
        throw LIQError("could not allocate palette");
    }

    // Write modified RGBA buffer into image

    lodepng::State state;
    lodepng_state_init(&state);
    state.info_raw.colortype = LCT_PALETTE;
    state.info_raw.bitdepth = 8;
    state.info_png.color.colortype = LCT_PALETTE;
    state.info_png.color.bitdepth = 8;

    for (int i = 0; i < palette->count; i++) {
        lodepng_palette_add(&state.info_png.color, palette->entries[i].r,
            palette->entries[i].g, palette->entries[i].b,
            palette->entries[i].a);
        lodepng_palette_add(&state.info_raw, palette->entries[i].r,
            palette->entries[i].g, palette->entries[i].b,
            palette->entries[i].a);
    }

    thumb->isPNG = true;
    free(thumb->img.data);
    thumb->img.data = NULL;
    thumb->img.size = 0;
    unsigned int status = lodepng_encode(&thumb->img.data, &thumb->img.size,
        (const unsigned char*)(void*)raw_out, thumb->img.width,
        thumb->img.height, &state);
    if (status) {
        throw LoadpngError(status);
    }
}

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
        img.magick("PNG");
        img.quality(105);
        thumb->isPNG = true;
    } else {
        img.magick("JPEG");
        if (opts.JPEGCompression <= 100) {
            img.quality(opts.JPEGCompression);
        }
    }

    Magick::Blob out;
    img.write(&out);
    thumb->img.data = (uint8_t*)malloc(out.length());
    memcpy(thumb->img.data, out.data(), out.length());
    thumb->img.size = out.length();

    // TODO: Fix this. Seems to be outputting only one channel or something
    // TODO: Convert to RGBA buffer not a PNG intermediary and avoid extra copy
    // if (need_png) {
    //     compress_png(thumb, img.gamma(), opts.PNGCompression);
    // }
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

    // Read only the first frame/page of GIFs and PDFs
    img.subImage(0);
    img.subRange(1);

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
        img.autoOrient();
    }
    // Strip EXIF metadata, if any
    img.strip();

    // Maintain aspect ratio
    const double scale = img.columns() >= img.rows()
        ? (double)img.columns() / (double)opts.thumbDims.width
        : (double)img.rows() / (double)opts.thumbDims.height;
    thumb->img.width = (unsigned long)((double)img.columns() / scale);
    thumb->img.height = (unsigned long)((double)img.rows() / scale);

    img.thumbnail(Magick::Geometry(thumb->img.width, thumb->img.height));
    write_thumb(img, thumb, opts);
}

// Catches amd converts exception, if any, to C string and returns it.
// Otherwise returns NULL.
static char* pass_exception(std::function<void()> fn)
{
    try {
        fn();
        return NULL;
    } catch (...) {
        auto e = std::current_exception();
        try {
            if (e) {
                std::rethrow_exception(e);
            }
            return NULL;
        } catch (const std::exception& e) {
            char* buf = (char*)malloc(strlen(e.what()) + 1);
            strcpy(buf, e.what());
            return buf;
        }
    }
}

extern "C" char* thumbnail(
    struct Buffer* src, struct Thumbnail* thumb, const struct Options opts)
{
    return pass_exception([=]() { _thumbnail(src, thumb, opts); });
}
