#include "thumbnailer.h"
#include <float.h>
#include <libavutil/imgutils.h>
#include <libswscale/swscale.h>

/**
 * Potential thumbnail lookup filter to reduce the risk of an inappropriate
 * selection (such as a black frame) we could get with an absolute seek.
 *
 * Simplified version of algorithm by Vadim Zaliva <lord@crocodile.org>.
 * http://notbrainsurgery.livejournal.com/29773.html
 *
 * Adapted by Janis Petersons <bakape@gmail.com>
 */

#define HIST_SIZE (3 * 256)
#define MAX_FRAMES 10

// Compute sum-square deviation to estimate "closeness"
static double compute_error(
    const int hist[HIST_SIZE], const double median[HIST_SIZE])
{
    double sum_sq_err = 0;
    for (int i = 0; i < HIST_SIZE; i++) {
        const double err = median[i] - (double)hist[i];
        sum_sq_err += err * err;
    }
    return sum_sq_err;
}

// Select best frame based on RGB histograms
static AVFrame* select_best_frame(AVFrame* frames[], int size)
{
    // RGB color distribution histograms of the frames
    int hists[MAX_FRAMES][HIST_SIZE] = { 0 };

    // Compute each frame's histogram
    for (int frame_i = 0; frame_i < size; frame_i++) {
        const AVFrame* f = frames[frame_i];
        uint8_t* p = f->data[0];
        for (int j = 0; j < f->height; j++) {
            for (int i = 0; i < f->width; i++) {
                hists[frame_i][p[i * 3]]++;
                hists[frame_i][256 + p[i * 3 + 1]]++;
                hists[frame_i][2 * 256 + p[i * 3 + 2]]++;
            }
            p += f->linesize[0];
        }
    }

    // Average all histograms
    double average[HIST_SIZE] = { 0 };
    for (int j = 0; j < size; j++) {
        for (int i = 0; i < size; i++) {
            average[j] = (double)hists[i][j];
        }
        average[j] /= size;
    }

    // Find the frame closer to the average using the sum of squared errors
    double min_sq_err = DBL_MAX;
    int best_i = 0;
    for (int i = 0; i < size; i++) {
        const double sq_err = compute_error(hists[i], average);
        if (sq_err < min_sq_err) {
            best_i = i;
            min_sq_err = sq_err;
        }
    }
    return frames[best_i];
}

// Calculate size and allocate buffer
static void alloc_buffer(struct Buffer* dst)
{
    dst->size
        = av_image_get_buffer_size(AV_PIX_FMT_RGBA, dst->width, dst->height, 1);
    dst->data = malloc(dst->size);
}

// Use point subsampling to scale image up to 4 times thumbnail size
static int upsample(struct Buffer* dst, const AVFrame const* frame)
{
    struct SwsContext* ctx
        = sws_getContext(frame->width, frame->height, frame->format, dst->width,
            dst->height, AV_PIX_FMT_RGBA, SWS_POINT, NULL, NULL, NULL);
    if (!ctx) {
        return AVERROR(ENOMEM);
    }

    alloc_buffer(dst);
    uint8_t* dst_data[1] = { dst->data }; // RGB have one plane
    int dst_linesize[1] = { 4 * dst->width }; // RGBA stride

    sws_scale(ctx, (const uint8_t* const*)frame->data, frame->linesize, 0,
        frame->height, dst_data, dst_linesize);

    sws_freeContext(ctx);
    return 0;
}

struct Pixel {
    // uint16_t fits the max value of 255 * 16
    uint16_t r, g, b, a;
};

// Downscale upsampled image
static void downscale(struct Buffer* dst, const struct Buffer const* src)
{
    alloc_buffer(dst);

    // First sum all pixels into a multidimensional array
    struct Pixel img[dst->height][dst->width];
    memset(img, 0, dst->height * dst->width * sizeof(struct Pixel));

    int i = 0;
    for (int y = 0; y < src->height; y++) {
        const int dest_y = y ? y / 4 : 0;
        for (int x = 0; x < src->width; x++) {
            struct Pixel* p = &img[dest_y][x ? x / 4 : 0];

            // Skip pixels with maxed transparency
            const uint8_t alpha = src->data[i + 3];
            if (alpha == 0) {
                p->a += alpha;
            } else {
                // Unrolled for less data dependency
                p->r += src->data[i];
                p->g += src->data[i + 1];
                p->b += src->data[i + 2];
                p->a += alpha;
            }

            i += 4; // Less data dependency than i++
        }
    }

    // Then average them and arrange as RGBA
    i = 0;
    for (int y = 0; y < dst->height; y++) {
        for (int x = 0; x < dst->width; x++) {
            const struct Pixel p = img[y][x];

            // Unrolled for less data dependency
            dst->data[i] = p.r ? p.r / 16 : 0;
            dst->data[i + 1] = p.g ? p.g / 16 : 0;
            dst->data[i + 2] = p.b ? p.b / 16 : 0;
            dst->data[i + 3] = p.a ? p.a / 16 : 0;
            i += 4;
        }
    }
}

// Encode and scale frame to RGBA image
static int encode_frame(
    struct Buffer* img, AVFrame* frame, const struct Dims thumb_dims)
{
    // Maintain aspect ratio
    double scale;
    if (frame->width >= frame->height) {
        scale = (double)(frame->width) / (double)(thumb_dims.width);
    } else {
        scale = (double)(frame->height) / (double)(thumb_dims.height);
    }
    img->width = (unsigned long)((double)frame->width / scale);
    img->height = (unsigned long)((double)frame->height / scale);

    // Subsample to 4 times the thumbnail size and then Box subsample that.
    // A decent enough compromise between quality and performance for images
    // around the thumbnail size and much bigger ones.
    struct Buffer enlarged
        = { .width = img->width * 4, .height = img->height * 4 };
    int err = upsample(&enlarged, frame);
    if (err) {
        return err;
    }
    downscale(img, &enlarged);
    free(enlarged.data);
    return err;
}

int generate_thumbnail(struct Buffer* img, AVFormatContext* avfc,
    AVCodecContext* avcc, const int stream, const struct Dims thumb_dims)
{
    int err = 0;
    int size = 0;
    int i = 0;
    AVPacket pkt;
    AVFrame* frames[MAX_FRAMES] = { NULL };
    AVFrame* next = NULL;

    // Read up to 10 frames in 10 frame intervals
    while (1) {
        err = av_read_frame(avfc, &pkt);
        switch (err) {
        case 0:
            break;
        case -1:
            // I don't know why, but this happens for some AVI and OGG files
            // mid-read. If some frames were actually read, just silence the
            // error and select from those.
            if (size) {
                err = 0;
            }
            goto end;
        default:
            goto end;
        }

        if (pkt.stream_index == stream) {
            err = avcodec_send_packet(avcc, &pkt);
            if (err < 0) {
                goto end;
            }

            if (!next) {
                next = av_frame_alloc();
                if (!next) {
                    err = AVERROR(ENOMEM);
                    goto end;
                }
            }
            err = avcodec_receive_frame(avcc, next);
            switch (err) {
            case 0:
                // Read only every 10th frame
                if (!(i++ % 10)) {
                    frames[size++] = next;
                    next = NULL;
                    if (size == MAX_FRAMES) {
                        goto end;
                    }
                } else {
                    av_frame_free(&next);
                    next = NULL;
                }
                break;
            case AVERROR(EAGAIN):
                break;
            default:
                goto end;
            }
        }
        av_packet_unref(&pkt);
    }

end:
    if (pkt.buf) {
        av_packet_unref(&pkt);
    }
    switch (err) {
    case AVERROR_EOF:
        err = 0;
    case 0:
        if (!size) {
            err = -1;
        } else {
            err = encode_frame(
                img, select_best_frame(frames, size), thumb_dims);
        }
        break;
    }
    for (int i = 0; i < size; i++) {
        av_frame_free(&frames[i]);
    }
    if (next) {
        av_frame_free(&next);
    }
    return err;
}
