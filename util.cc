#include "util.hh"
#include <cstring>

uint8_t get_quality(uint8_t def, uint8_t q)
{
    if (q && q <= 100) {
        return q;
    }
    return def;
}

char* pass_exception(std::function<char*()> fn)
{
    try {
        return fn();
    } catch (...) {
        auto e = std::current_exception();
        try {
            if (e) {
                std::rethrow_exception(e);
            }
            return NULL;
        } catch (const std::exception& e) {
            char* buf = (char*)malloc(strlen(e.what()) + 1);
            strcpy(buf, e.what());
            return buf;
        }
    }
}
