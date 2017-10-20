#include "video.h"
#include <libavfilter/avfilter.h>
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
#define MAX_FRAMES 100

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

// RGB color distribution histograms of the frames. Reused between all calls to
// avoid allocations. Need lock of hist_mu.
static int hists[MAX_FRAMES][HIST_SIZE];
pthread_mutex_t hist_mu = PTHREAD_MUTEX_INITIALIZER;

// Select best frame based on RGB histograms
static int select_best_frame(AVFrame* frames[], int* best_i)
{
    int err = pthread_mutex_lock(&hist_mu);
    if (err) {
        return err;
    }

    // First rezero after last use
    memset(hists, 0, sizeof(int) * MAX_FRAMES * HIST_SIZE);

    // Compute each frame's histograms
    int frame_i;
    for (frame_i = 0; frame_i < MAX_FRAMES; frame_i++) {
        const AVFrame* f = frames[frame_i];
        uint8_t* p = f->data[0];
        if (!f) {
            frame_i--;
            break;
        }
        for (int j = 0; j < f->height; j++) {
            for (int i = 0; f->width; i++) {
                for (int k = 0; k < 3; k++) {
                    hists[frame_i][k * 256 + p[i * 3 + k]]++;
                }
            }
            p += f->linesize[0];
        }
    }

    // Average histograms of up to 100 frames
    double average[HIST_SIZE] = { 0 };
    for (int j = 0; j <= frame_i; j++) {
        for (int i = 0; i <= frame_i; i++) {
            average[j] = (double)hists[i][j];
        }
        average[j] /= frame_i + 1;
    }

    // Find the frame closer to the average using the sum of squared errors
    double min_sq_err = -1;
    for (int i = 0; i <= frame_i; i++) {
        const double sq_err = compute_error(hists[i], average);
        if (i == 0 || sq_err < min_sq_err) {
            *best_i = i;
            min_sq_err = sq_err;
        }
    }

    return pthread_mutex_unlock(&hist_mu);
}

// Encode frame to RGBA image
static int encode_frame(struct Buffer* img, const AVFrame const* frame)
{
    struct SwsContext* ctx;
    uint8_t* dst_data[1];
    int dst_linesize[1];

    ctx = sws_getContext(frame->width, frame->height, frame->format,
        frame->width, frame->height, AV_PIX_FMT_RGBA,
        SWS_BICUBIC | SWS_ACCURATE_RND, NULL, NULL, NULL);
    if (!ctx) {
        return -1;
    }

    img->width = (unsigned long)frame->width;
    img->height = (unsigned long)frame->height;
    img->size = (size_t)av_image_get_buffer_size(
        AV_PIX_FMT_RGBA, frame->width, frame->height, 1);
    img->data = dst_data[0] = malloc(img->size); // RGB have one plane
    dst_linesize[0] = 4 * img->width; // RGBA stride

    sws_scale(ctx, (const uint8_t* const*)frame->data, frame->linesize, 0,
        frame->height, (uint8_t * const*)dst_data, dst_linesize);

    sws_freeContext(ctx);
    return 0;
}

int extract_video_image(struct Buffer* img, AVFormatContext* avfc,
    AVCodecContext* avcc, const int stream)
{
    int err = 0;
    int best_frame = 0;
    int frame_i = 0;
    AVPacket pkt;
    AVFrame* frames[MAX_FRAMES] = { 0 };

    // Read up to 100 frames
    while (1) {
        err = av_read_frame(avfc, &pkt);
        if (err < 0) {
            goto end;
        }

        if (pkt.stream_index == stream) {
            err = avcodec_send_packet(avcc, &pkt);
            if (err < 0) {
                goto end;
            }

            if (!frames[frame_i]) {
                frames[frame_i] = av_frame_alloc();
            }
            err = avcodec_receive_frame(avcc, frames[frame_i]);
            switch (err) {
            case 0:
                if (++frame_i == MAX_FRAMES) {
                    goto end;
                }
                av_packet_unref(&pkt);
                continue;
            case AVERROR(EAGAIN):
                av_packet_unref(&pkt);
                continue;
            default:
                goto end;
            }
        }
    }

end:
    switch (err) {
    case AVERROR_EOF:
        err = 0;
    case 0:
        err = select_best_frame(frames, &best_frame);
        if (!err) {
            err = encode_frame(img, frames[best_frame]);
        }
        break;
    }
    av_packet_unref(&pkt);
    for (int i = 0; i < MAX_FRAMES; i++) {
        if (!frames[i]) {
            break;
        }
        av_frame_free(&frames[i]);
    }
    return err;
}
