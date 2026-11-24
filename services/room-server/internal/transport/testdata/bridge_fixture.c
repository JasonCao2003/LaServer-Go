/* ABI fixture only: this is not an ENet implementation. */
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
struct event { uint64_t peer, channel, flags, length; };
struct host { uint8_t data[1024]; size_t length; uint32_t flags; uint8_t channel; int pending; };
uint32_t skyla_enet_abi(void) {
#ifdef BAD_ABI
    return 99;
#else
    return 1;
#endif
}
void *skyla_enet_open(uint32_t ip, uint16_t port, size_t peers, size_t frame) {
    if (ip != 0x7f000001 || port != 23456 || peers != 2 || frame != 1024) return NULL;
    return calloc(1, sizeof(struct host));
}
int32_t skyla_enet_poll(struct host *h, uint32_t timeout, struct event *e, uint8_t *buf, size_t length) {
    (void)timeout;
#ifdef BAD_LENGTH
    e->length = length + 1;
    return 3;
#endif
#ifdef BAD_KIND
    return -2;
#endif
    if (!h->pending) return 0;
    *e = (struct event){7, h->channel, h->flags, h->length};
    memcpy(buf, h->data, h->length);
    h->pending = 0;
    return 3;
}
#ifndef MISSING_SEND
int32_t skyla_enet_send(struct host *h, size_t peer, uint8_t channel, const uint8_t *buf, size_t length, uint32_t reliable) {
    if (peer != 7 || channel > 1 || length > sizeof(h->data)) return -1;
    if (length) memcpy(h->data, buf, length);
    h->length = length; h->channel = channel; h->flags = reliable; h->pending = 1;
    return 0;
}
#endif
void skyla_enet_disconnect(struct host *h, size_t peer, uint32_t code) {
    (void)peer; (void)code; h->pending = 0;
}
void skyla_enet_close(struct host *h) { free(h); }
