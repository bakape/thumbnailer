#ifndef MIT_LICENSE

// GraphicsMagick does not support PNG8 properly, so use lodepng instead
#include "lodepng.h"

#include "compress_png.h"
#include "libimagequant.h"
#include "thumbnailer.h"
#include <string.h>

static char* format_error(const char* prefix, const char* err)
{
    char* b = malloc(1024);
    snprintf(b, 1024, "%s: %s", prefix, err);
    return b;
}

static char* format_liq_error(int code)
{
    char* err;
    switch (code) {
    case LIQ_QUALITY_TOO_LOW:
        err = "quality too low";
    case LIQ_VALUE_OUT_OF_RANGE:
        err = "value out of range";
    case LIQ_OUT_OF_MEMORY:
        err = "out of memory";
    case LIQ_ABORTED:
        err = "aborted";
    case LIQ_BITMAP_NOT_AVAILABLE:
        err = "bitmap not available";
    case LIQ_BUFFER_TOO_SMALL:
        err = "buffer too small";
    case LIQ_INVALID_POINTER:
        err = "invalid pointer";
    case LIQ_UNSUPPORTED:
        err = "unsupported";
    default:
        err = "unknown error";
    }

    return format_error("imagequant", err);
}

char* format_loadpng_error(int code)
{
    return format_error("loadpng", lodepng_error_text(code));
}

// Write Image as RGBA buffer
static char* image_to_raw(Image* img, uint8_t** buf, size_t* size)
{
    ImageInfo* info = CloneImageInfo(NULL);
    ExceptionInfo ex;
    GetExceptionInfo(&ex);

    strcpy(info->magick, "RGBA");
    info->depth = 8;
    *buf = ImageToBlob(info, img, size, &ex);

    DestroyImageInfo(info);
    char* err = format_magick_exception(&ex);
    DestroyExceptionInfo(&ex);
    return err;
}

static char* encode(uint8_t* in, size_t in_size, const liq_palette* palette,
    struct Thumbnail* thumb)
{
    LodePNGState state;

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

    unsigned int err = lodepng_encode(&thumb->img.data, &thumb->img.size, in,
        thumb->img.width, thumb->img.height, &state);
    lodepng_state_cleanup(&state);
    if (err) {
        return format_loadpng_error(err);
    }
    return NULL;
}

char* compress_png(
    Image* img, struct Thumbnail* thumb, const struct CompressionRange quality)
{
    const unsigned width = img->columns, height = img->rows;
    char* err = NULL;
    int err_code = 0;
    liq_attr* handle = NULL;
    liq_image* input_image = NULL;
    liq_result* res = NULL;
    uint8_t *raw_in = NULL, *raw_out = NULL;

#define HANDLE_LIQ_ERROR                                                       \
    if (err_code) {                                                            \
        err = format_liq_error(err_code);                                      \
        goto end;                                                              \
    }

    // Prepare handle
    handle = liq_attr_create();
    if (!handle) {
        err = format_error("imagequant", "can not run on this machine");
        goto end;
    }
    err_code = liq_set_quality(
        handle, get_quality(10, quality.min), get_quality(100, quality.max));
    HANDLE_LIQ_ERROR;

    // Read image into libimagequant
    size_t raw_in_size = 0;
    err = image_to_raw(img, &raw_in, &raw_in_size);
    if (err) {
        goto end;
    }
    input_image
        = liq_image_create_rgba(handle, raw_in, width, height, img->gamma);
    if (!input_image) {
        err = format_error("imagequant", "error allocating image");
        goto end;
    }

    // Quantize image
    err_code = liq_image_quantize(input_image, handle, &res);
    HANDLE_LIQ_ERROR;

    // Write image to RGBA buffer
    err_code = liq_set_dithering_level(res, 1.0);
    HANDLE_LIQ_ERROR;
    const size_t pixels_size = width * height;
    raw_out = malloc(pixels_size);
    err_code = liq_write_remapped_image(res, input_image, raw_out, pixels_size);
    HANDLE_LIQ_ERROR;
    const liq_palette* palette = liq_get_palette(res);
    if (!palette) {
        err = format_error("imagequant", "could not get palette");
        goto end;
    }

    // Write modified RGBA buffer into image
    err = encode(raw_out, pixels_size, palette, thumb);

end:
    if (handle) {
        liq_attr_destroy(handle);
    }
    if (input_image) {
        liq_image_destroy(input_image);
    }
    if (res) {
        liq_result_destroy(res);
    }
    if (raw_in) {
        free(raw_in);
    }
    if (raw_out) {
        free(raw_out);
    }
    return err;
}

#endif // MIT_LICENSE
