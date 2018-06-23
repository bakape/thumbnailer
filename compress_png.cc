#ifndef MIT_LICENSE

// GraphicsMagick does not support PNG8 properly, so use lodepng instead
#include "lodepng.h"
// Because you can't include C++ files in Go files and lodepng is in another dir
#include "lodepng.cpp"

extern "C" {
#include "libimagequant.h"
#include "thumbnailer.h"
#include <string.h>
}
#include "compress_png.hh"
#include "util.hh"

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

void compress_png(Magick::Image& img, struct Thumbnail* thumb,
    const struct CompressionRange quality)
{
    const unsigned width = img.columns(), height = img.rows();

    // Prepare handle
    RAII<liq_attr, liq_attr_destroy> handle(liq_attr_create());
    if (!handle) {
        throw LIQError("can not run on this machine");
    }
    int err = liq_set_quality(
        handle, get_quality(10, quality.min), get_quality(100, quality.max));
    if (err) {
        throw LIQError(err);
    }

    // Read image into libimagequant
    Magick::Blob raw_in;
    img.magick("RGBA");
    img.depth(8);
    img.write(&raw_in);
    RAII<liq_image, liq_image_destroy> input_image(liq_image_create_rgba(
        handle, raw_in.data(), width, height, img.gamma()));
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
    const size_t pixels_size = width * height;
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

    unsigned int status = lodepng_encode(&thumb->img.data, &thumb->img.size,
        (const unsigned char*)(void*)raw_out, width, height, &state);
    if (status) {
        throw LoadpngError(status);
    }
}

#endif // MIT_LICENSE
