#pragma once

#include <stdint.h>

// Check, if quality is set and valid, or return default
uint8_t get_quality(uint8_t def, uint8_t q);

// Creates RAII wrapper for C type T with simple destructor D.
template <class T, void (*D)(T*)> class RAII {
public:
    RAII(T* ptr = nullptr)
        : ptr(ptr)
    {
    }

    ~RAII()
    {
        if (ptr) {
            D(ptr);
        }
    }

    // Implicit conversion to internal pointer
    operator T*() { return ptr; }

    // Override & for taking a pointer of the internal pointer
    T** operator&() { return &ptr; }

private:
    T* ptr = nullptr;
};
