#include "util.hh"

uint8_t get_quality(uint8_t def, uint8_t q)
{
    if (q && q <= 100) {
        return q;
    }
    return def;
}
