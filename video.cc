extern "C" {
#include "video.h"
#include <libavutil/imgutils.h>
#include <libswscale/swscale.h>
}
#include "util.hh"
#include <array>
#include <utility>
#include <vector>

/**
 * Potential thumbnail lookup filter to reduce the risk of an inappropriate
 * selection (such as a black frame) we could get with an absolute seek.
 *
 * Simplified version of algorithm by Vadim Zaliva <lord@crocodile.org>.
 * http://notbrainsurgery.livejournal.com/29773.html
 *
 * Adapted by Janis Petersons <bakape@gmail.com>
 */

const unsigned HIST_SIZE = 3 * 256, MAX_FRAMES = 100;

class Frame : public RAIIPP<AVFrame, av_frame_free> {
public:
    Frame()
        : RAIIPP(av_frame_alloc())
    {
    }

    Frame(Frame&& rhs)
    {
        ptr = rhs.ptr;
        rhs.ptr = NULL;
    }
};

typedef std::array<int, HIST_SIZE> histogram;

// Compute sum-square deviation to estimate "closeness"
static double compute_error(
    const histogram& hist, const std::vector<double> average)
{
    double sum_sq_err = 0;
    for (int i = 0; i < average.size(); i++) {
        const double err = average[i] - (double)hist[i];
        sum_sq_err += err * err;
    }
    return sum_sq_err;
}

// Select best frame based on RGB histograms
static int select_best_frame(const std::vector<Frame>& frames)
{
    // RGB color distribution histograms of the frames
    std::vector<histogram> hists(frames.size());

    // Compute each frame's histogram
    for (int i = 0; i < frames.size(); i++) {
        const AVFrame* f = frames[i].ptr;
        uint8_t* p = f->data[0];
        for (int j = 0; j < f->height; j++) {
            for (int k = 0; i < f->width; k++) {
                for (int l = 0; l < 3; l++) {
                    hists[i][l * 256 + p[k * 3 + l]]++;
                }
            }
            p += f->linesize[0];
        }
    }

    // Average histograms of up to 100 frames
    std::vector<double> average(frames.size());
    for (int i = 0; i < frames.size(); i++) {
        for (int j = 0; j < frames.size(); j++) {
            average[i] = (double)hists[j][i];
        }
        average[i] /= frames.size() + 1;
    }

    // Find the frame closer to the average using the sum of squared errors
    int best_i = 0;
    double min_sq_err = -1;
    for (int i = 0; i < frames.size(); i++) {
        const double sq_err = compute_error(hists[i], average);
        if (i == 0 || sq_err < min_sq_err) {
            best_i = i;
            min_sq_err = sq_err;
        }
    }
    return best_i;
}

// Encode frame to RGBA image
static char* encode_frame(struct Buffer* img, const AVFrame* f)
{
    struct SwsContext* ctx;
    uint8_t* dst_data[1];
    int dst_linesize[1];

    ctx = sws_getContext(f->width, f->height,
        static_cast<AVPixelFormat>(f->format), f->width, f->height,
        AV_PIX_FMT_RGBA, SWS_BICUBIC | SWS_ACCURATE_RND, NULL, NULL, NULL);
    if (!ctx) {
        return format_error(AVERROR(ENOMEM));
    }

    img->width = (unsigned long)f->width;
    img->height = (unsigned long)f->height;
    img->size = (size_t)av_image_get_buffer_size(
        AV_PIX_FMT_RGBA, f->width, f->height, 1);
    img->data = dst_data[0] = (uint8_t*)malloc(img->size); // RGB have one plane
    dst_linesize[0] = 4 * img->width; // RGBA stride

    sws_scale(ctx, (const uint8_t* const*)f->data, f->linesize, 0, f->height,
        (uint8_t * const*)dst_data, dst_linesize);

    sws_freeContext(ctx);
    return NULL;
}

// Packet that dereferences itself on destruction
class AVPacketRef : public AVPacket {
public:
    ~AVPacketRef()
    {
        if (buf) {
            av_packet_unref(this);
        }
    }
};

// Read up to 100 frames. It is possible for no frames and a NUL error to be
// returned.
static char* read_frames(std::vector<Frame>& frames, AVFormatContext* avfc,
    AVCodecContext* avcc, const int stream)
{
    frames.reserve(MAX_FRAMES);
    Frame next;

    while (1) {
        AVPacketRef pkt;
        int err = av_read_frame(avfc, &pkt);
        switch (err) {
        case 0:
            break;
        case -1:
        case AVERROR_EOF:
            // I don't know why, but this happens for some AVI and OGG files
            // mid-read. No descriptive error is putput. If some frames were
            // actually read, just silence the error and select from those.
            return NULL;
        default:
            return format_error(err);
        }

        if (pkt.stream_index == stream) {
            err = avcodec_send_packet(avcc, &pkt);
            switch (err) {
            case 0:
                break;
            case AVERROR_EOF:
                return NULL;
            default:
                return format_error(err);
            }

            err = avcodec_receive_frame(avcc, next);
            switch (err) {
            case 0:
                frames.push_back(std::move(next));
                if (frames.size() == MAX_FRAMES) {
                    return NULL;
                }
                break;
            case AVERROR(EAGAIN):
                break;
            case AVERROR_EOF:
                return NULL;
            default:
                return format_error(err);
            }
        }
    }
}

static char* malloc_string(const char* s)
{
    return strcpy((char*)malloc((strlen(s) + 1) * sizeof(char)), s);
}

static char* _extract_video_image(struct Buffer* img, AVFormatContext* avfc,
    AVCodecContext* avcc, const int stream)
{
    std::vector<Frame> frames;
    auto err = read_frames(frames, avfc, avcc, stream);
    if (err) {
        return err;
    }
    if (!frames.size()) {
        return malloc_string("no video frames decoded");
    }
    return encode_frame(img, frames[select_best_frame(frames)]);
}

extern "C" char* extract_video_image(struct Buffer* img, AVFormatContext* avfc,
    AVCodecContext* avcc, const int stream)
{
    return pass_exception(
        [=]() { return _extract_video_image(img, avfc, avcc, stream); });
}
