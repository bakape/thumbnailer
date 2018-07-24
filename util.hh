#pragma once

#include <functional>
#include <stdint.h>

// Check, if quality is set and valid, or return default
uint8_t get_quality(uint8_t def, uint8_t q);

// Base class for storing C type pointer in C++ type with RAII capabilities
template <class T> class RAIIBase {
public:
    RAIIBase(T* ptr = nullptr)
        : ptr(ptr)
    {
    }

    // Implicit conversion to internal pointer
    operator T*() { return ptr; }

    // Override & for taking a pointer of the internal pointer
    T** operator&() { return &ptr; }

    T* ptr = nullptr;
};

// Creates RAII wrapper for C type T pointer with simple destructor D
template <class T, void (*D)(T*)> class RAII : public RAIIBase<T> {
public:
    using RAIIBase<T>::RAIIBase;
    using RAIIBase<T>::ptr;

    ~RAII()
    {
        if (ptr) {
            D(ptr);
        }
    }
};

// As RAII, but the destructor takes a pointer of a pointer
template <class T, void (*D)(T**)> class RAIIPP : public RAIIBase<T> {
public:
    using RAIIBase<T>::RAIIBase;
    using RAIIBase<T>::ptr;

    ~RAIIPP()
    {
        if (ptr) {
            D(&ptr);
        }
    }
};

// Catches and converts exception, if any, to C string and returns it.
// Otherwise returns fn().
char* pass_exception(std::function<char*()> fn);
