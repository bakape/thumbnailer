#pragma once
#include <magick/api.h>
#include <stddef.h>
#include <stdint.h>

struct Buffer {
    uint8_t* data;
    size_t size;
    unsigned long width, height;
};

struct Dims {
    unsigned long width, height;
};

// Check, if quality is set and valid, or return default
uint8_t get_quality(uint8_t def, uint8_t q);

char* copy_string(const char* s);

// Format graphicsmagick exception as error string
char* format_magick_exception(const ExceptionInfo* ex);
