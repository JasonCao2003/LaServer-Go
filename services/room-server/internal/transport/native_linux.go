//go:build linux && cgo

// Linux 原生 UDP 传输实现

package transport

/*
#cgo LDFLAGS: -ldl
#include "native_linux.h"
*/
import "C"

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net/netip"
	"path/filepath"
	"strings"
	"time"
	"unsafe"
)

const (
	maxENetPeers   = 0xfff
	maxNativeFrame = 32 << 20
	reliableFlag   = 1
)

// Native owns a dynamically loaded ABI v1 SkyENet host. All methods, including
// Close, must be called by its owner loop. Poll returns owned packet data.
// The C bridge may use Go byte buffers only for the duration of each call.
type Native struct {
	library *C.struct_skyla_library
	buffer  []byte
}

// 打开原生 UDP 传输
func OpenNative(path string, addr netip.AddrPort, maxPeers, maxFrame int) (*Native, error) {
	if !filepath.IsAbs(path) || strings.ContainsRune(path, 0) {
		return nil, errors.New("SkyENet bridge path must be absolute and contain no NUL")
	}
	if !addr.Addr().Is4() {
		return nil, errors.New("SkyENet adapter requires IPv4")
	}
	if maxPeers < 1 || maxPeers > maxENetPeers {
		return nil, fmt.Errorf("SkyENet peer limit must be between 1 and %d", maxENetPeers)
	}
	if maxFrame < 1 || maxFrame > maxNativeFrame {
		return nil, fmt.Errorf("SkyENet frame limit must be between 1 and %d", maxNativeFrame)
	}
	name := C.CString(path)
	defer C.free(unsafe.Pointer(name))
	var detail [1024]C.char
	ip := addr.Addr().As4()
	library := C.skyla_load(name, C.uint32_t(binary.BigEndian.Uint32(ip[:])), C.uint16_t(addr.Port()), C.size_t(maxPeers), C.size_t(maxFrame), &detail[0], C.size_t(len(detail)))
	if library == nil {
		return nil, fmt.Errorf("%s (address %s)", C.GoString(&detail[0]), addr)
	}
	return &Native{library: library, buffer: make([]byte, maxFrame)}, nil
}

// 轮询传输事件
func (host *Native) Poll(timeout time.Duration) (Event, bool, error) {
	if host.library == nil {
		return Event{}, false, errors.New("SkyENet host is closed")
	}
	if timeout < 0 || timeout > time.Duration(^uint32(0))*time.Millisecond {
		return Event{}, false, errors.New("SkyENet poll timeout is out of range")
	}
	var received C.struct_skyla_native_event
	kind := int32(C.skyla_poll(host.library, C.uint32_t(timeout/time.Millisecond), &received, (*C.uint8_t)(unsafe.Pointer(unsafe.SliceData(host.buffer))), C.size_t(len(host.buffer))))
	switch kind {
	case 0:
		return Event{}, false, nil
	case int32(Connected), int32(Disconnected), int32(Received):
		if uint64(received.data_length) > uint64(len(host.buffer)) {
			return Event{}, false, errors.New("SkyENet bridge returned an invalid packet length")
		}
		return Event{Kind: EventKind(kind), Peer: uint64(received.peer), Channel: uint8(received.channel), Reliable: received.flags&reliableFlag != 0, Data: append([]byte(nil), host.buffer[:int(received.data_length)]...)}, true, nil
	default:
		return Event{}, false, fmt.Errorf("poll SkyENet host: native result %d", kind)
	}
}

// 发送数据到指定连接
func (host *Native) Send(peer uint64, channel uint8, reliable bool, data []byte) error {
	if host.library == nil {
		return errors.New("SkyENet host is closed")
	}
	var flag C.uint32_t
	if reliable {
		flag = 1
	}
	result := C.skyla_send(host.library, C.size_t(peer), C.uint8_t(channel), (*C.uint8_t)(unsafe.Pointer(unsafe.SliceData(data))), C.size_t(len(data)), flag)
	if result != 0 {
		return fmt.Errorf("send to SkyENet peer %d on channel %d", peer, channel)
	}
	return nil
}

// 断开指定连接
func (host *Native) Disconnect(peer uint64, code uint32) {
	if host.library != nil {
		C.skyla_disconnect(host.library, C.size_t(peer), C.uint32_t(code))
	}
}

// 关闭传输资源
func (host *Native) Close() error {
	if host.library == nil {
		return nil
	}
	result := C.skyla_close(host.library)
	host.library = nil
	if result != 0 {
		return errors.New("unload SkyENet bridge failed")
	}
	return nil
}
