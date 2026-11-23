#include <enet/enet.h>
#include <stdint.h>
#include <string.h>

#if defined(_WIN32)
#define SKYLA_EXPORT __declspec(dllexport)
#else
#define SKYLA_EXPORT __attribute__((visibility("default")))
#endif

struct skyla_event {
    uint64_t peer;
    uint64_t channel;
    uint64_t flags;
    uint64_t data_length;
};

SKYLA_EXPORT uint32_t skyla_enet_abi(void) {
    return 1;
}

SKYLA_EXPORT ENetHost *skyla_enet_open(uint32_t ipv4, uint16_t port,
                                     size_t max_peers, size_t max_frame) {
    if (enet_initialize() != 0) {
        return NULL;
    }
    ENetAddress address = {0};
    address.type = ENET_ADDRESS_TYPE_IPV4;
    address.port = port;
    for (int i = 0; i < 4; ++i) {
        address.host.v4[i] = (enet_uint8)(ipv4 >> ((3 - i) * 8));
    }
    ENetHost *host = enet_host_create(address.type, &address, max_peers, 2, 0, 0);
    if (host == NULL) {
        enet_deinitialize();
        return NULL;
    }
    host->checksum = enet_crc32;
    host->maximumPacketSize = max_frame;
    return host;
}

SKYLA_EXPORT int32_t skyla_enet_poll(ENetHost *host, uint32_t timeout,
                                   struct skyla_event *output, uint8_t *buffer,
                                   size_t buffer_length) {
    ENetEvent event = {0};
    int result = enet_host_service(host, &event, timeout);
    if (result <= 0) {
        return result;
    }
    output->peer = (uint64_t)(event.peer - host->peers);
    output->channel = 0;
    output->flags = 0;
    output->data_length = 0;
    if (event.type == ENET_EVENT_TYPE_RECEIVE) {
        ENetPacket *packet = event.packet;
        if (packet->dataLength > buffer_length) {
            enet_packet_destroy(packet);
            return -2;
        }
        output->channel = event.channelID;
        output->flags = packet->flags;
        output->data_length = packet->dataLength;
        if (packet->dataLength != 0) {
            memcpy(buffer, packet->data, packet->dataLength);
        }
        enet_packet_destroy(packet);
    }
    return (int32_t)event.type;
}

SKYLA_EXPORT int32_t skyla_enet_send(ENetHost *host, size_t peer, uint8_t channel,
                                   const uint8_t *buffer, size_t length, uint32_t reliable) {
    if (peer >= host->peerCount || channel >= host->channelLimit ||
        length > host->maximumPacketSize) {
        return -1;
    }
    ENetPacket *packet = enet_packet_create(buffer, length,
        reliable ? ENET_PACKET_FLAG_RELIABLE : 0);
    if (packet == NULL) {
        return -1;
    }
    int result = enet_peer_send(&host->peers[peer], channel, packet);
    if (result != 0) {
        enet_packet_destroy(packet);
    }
    return result;
}

SKYLA_EXPORT void skyla_enet_disconnect(ENetHost *host, size_t peer, uint32_t code) {
    if (peer < host->peerCount) {
        enet_peer_disconnect(&host->peers[peer], code);
    }
}

SKYLA_EXPORT void skyla_enet_close(ENetHost *host) {
    enet_host_destroy(host);
    enet_deinitialize();
}
