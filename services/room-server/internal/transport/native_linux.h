#ifndef SKYLA_NATIVE_LINUX_H
#define SKYLA_NATIVE_LINUX_H
#include <dlfcn.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

struct skyla_native_event { uint64_t peer, channel, flags, data_length; };
struct skyla_library {
    void *library, *host;
    int32_t (*poll)(void *, uint32_t, struct skyla_native_event *, uint8_t *, size_t);
    int32_t (*send)(void *, size_t, uint8_t, const uint8_t *, size_t, uint32_t);
    void (*disconnect)(void *, size_t, uint32_t);
    void (*close)(void *);
};

static struct skyla_library *skyla_load(const char *path, uint32_t ip, uint16_t port,
                                      size_t peers, size_t frame, char *error, size_t capacity) {
    struct skyla_library *lib = calloc(1, sizeof(*lib));
    if (!lib) { snprintf(error, capacity, "allocate native loader"); return NULL; }
    lib->library = dlopen(path, RTLD_NOW | RTLD_LOCAL);
    if (!lib->library) { snprintf(error, capacity, "load SkyENet bridge: %s", dlerror()); free(lib); return NULL; }
    uint32_t (*abi)(void);
    void *(*open_host)(uint32_t, uint16_t, size_t, size_t);
#define SKYLA_RESOLVE(dest, name) do { \
    dlerror(); \
    *(void **)(&(dest)) = dlsym(lib->library, name); \
    const char *detail = dlerror(); \
    if (detail || !(dest)) { \
        snprintf(error, capacity, "resolve SkyENet bridge %s: %s", name, detail ? detail : "null symbol"); \
        goto failure; \
    } \
} while (0)
    SKYLA_RESOLVE(abi, "skyla_enet_abi");
    SKYLA_RESOLVE(open_host, "skyla_enet_open");
    SKYLA_RESOLVE(lib->poll, "skyla_enet_poll");
    SKYLA_RESOLVE(lib->send, "skyla_enet_send");
    SKYLA_RESOLVE(lib->disconnect, "skyla_enet_disconnect");
    SKYLA_RESOLVE(lib->close, "skyla_enet_close");
#undef SKYLA_RESOLVE
    uint32_t version = abi();
    if (version != 1) { snprintf(error, capacity, "unsupported SkyENet bridge ABI %u", version); goto failure; }
    lib->host = open_host(ip, port, peers, frame);
    if (!lib->host) { snprintf(error, capacity, "open SkyENet host failed"); goto failure; }
    return lib;
failure:
    dlclose(lib->library);
    free(lib);
    return NULL;
}
static int32_t skyla_poll(struct skyla_library *lib, uint32_t timeout, struct skyla_native_event *event, uint8_t *buffer, size_t length) {
    return lib->poll(lib->host, timeout, event, buffer, length);
}
static int32_t skyla_send(struct skyla_library *lib, size_t peer, uint8_t channel, const uint8_t *buffer, size_t length, uint32_t reliable) {
    return lib->send(lib->host, peer, channel, buffer, length, reliable);
}
static void skyla_disconnect(struct skyla_library *lib, size_t peer, uint32_t code) {
    lib->disconnect(lib->host, peer, code);
}
static int skyla_close(struct skyla_library *lib) {
    lib->close(lib->host);
    int result = dlclose(lib->library);
    free(lib);
    return result;
}
#endif
