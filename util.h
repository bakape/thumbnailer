#pragma once
#include <stddef.h>
#include <stdint.h>

struct Buffer {
	uint8_t *data;
	size_t size;
	unsigned long width, height;
};

struct Dims {
	unsigned long width, height;
};
