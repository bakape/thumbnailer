#include "init.h"
#include <magick/api.h>
#include <signal.h>

#ifndef _WIN32
// Add the SA_ONSTACK flag to a listened on signal to play nice with the Go
// runtime
static void fixSignal(int signum)
{
    struct sigaction st;
    if (sigaction(signum, NULL, &st) < 0) {
        return;
    }
    st.sa_flags |= 0x08000000;
    sigaction(signum, &st, NULL);
}
#endif

void magickInit()
{
    InitializeMagick(NULL);

#ifndef _WIN32

#if defined(SIGINT)
    fixSignal(SIGINT);
#endif
#if defined(SIGTERM)
    fixSignal(SIGTERM);
#endif

#endif
}
