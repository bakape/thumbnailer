#include "meta.h"

// Find artist and title meta info if present
struct Meta retrieve_meta(AVFormatContext* ctx)
{
    AVDictionary* meta = ctx->metadata;
    AVDictionaryEntry* tag;
    struct Meta meta_out = { .title = NULL, .artist = NULL };
    if (!meta) {
        return meta_out;
    }

    tag = av_dict_get(meta, "title", NULL, 0);
    if (tag != NULL) {
        meta_out.title = tag->value;
    }
    tag = av_dict_get(meta, "artist", NULL, 0);
    if (tag != NULL) {
        meta_out.artist = tag->value;
    }

    return meta_out;
}
