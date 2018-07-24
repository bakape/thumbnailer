#include "util.h"
#include <string.h>

uint8_t get_quality(uint8_t def, uint8_t q)
{
    if (q && q <= 100) {
        return q;
    }
    return def;
}

char* copy_string(const char* s)
{
    char* b = malloc((strlen(s) + 1) * sizeof(char));
    return strcpy(b, s);
}

char* format_magick_exception(const ExceptionInfo* ex)
{
    if (!ex->reason && !ex->description) {
        return NULL;
    }
    char* b = malloc(1024);
    snprintf(b, 1024, "graphicsmagick: %s: %s", ex->reason, ex->description);
    return b;
}
