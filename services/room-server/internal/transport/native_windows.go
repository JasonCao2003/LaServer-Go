//go:build windows && amd64

package transport

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net/netip"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

const (
	bridgeABI      = 1
	maxENetPeers   = 0xfff
	maxNativeFrame = 32 << 20
	reliableFlag   = 1
)

type nativeEvent struct {
	peer       uint64
	channel    uint64
	flags      uint64
	dataLength uint64
}

// Native is a bound SkyENet host. Calls, including Close, belong to its
// owner loop. Poll returns owned packet data that survives the next call.
type Native struct {
	dll        *syscall.DLL
	handle     uintptr
	poll       *syscall.Proc
	send       *syscall.Proc
	disconnect *syscall.Proc
	close      *syscall.Proc
	buffer     []byte
}

func OpenNative(path string, addr netip.AddrPort, maxPeers, maxFrame int) (*Native, error) {
	if !filepath.IsAbs(path) {
		return nil, errors.New("SkyENet bridge path must be absolute")
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
	dll, err := syscall.LoadDLL(path)
	if err != nil {
		return nil, fmt.Errorf("load SkyENet bridge: %w", err)
	}
	host, err := openBridge(dll, addr, maxPeers, maxFrame)
	if err != nil {
		return nil, errors.Join(err, dll.Release())
	}
	return host, nil
}

func openBridge(dll *syscall.DLL, addr netip.AddrPort, maxPeers, maxFrame int) (*Native, error) {
	host := &Native{dll: dll}
	var abi, open *syscall.Proc
	for _, symbol := range []struct {
		name string
		dest **syscall.Proc
	}{
		{"skyla_enet_abi", &abi},
		{"skyla_enet_open", &open},
		{"skyla_enet_poll", &host.poll},
		{"skyla_enet_send", &host.send},
		{"skyla_enet_disconnect", &host.disconnect},
		{"skyla_enet_close", &host.close},
	} {
		proc, err := dll.FindProc(symbol.name)
		if err != nil {
			return nil, fmt.Errorf("resolve SkyENet bridge: %w", err)
		}
		*symbol.dest = proc
	}
	version, _, _ := abi.Call()
	if uint32(version) != bridgeABI {
		return nil, fmt.Errorf("unsupported SkyENet bridge ABI %d", uint32(version))
	}
	ip := addr.Addr().As4()
	ipv4 := binary.BigEndian.Uint32(ip[:])
	handle, _, _ := open.Call(uintptr(ipv4), uintptr(addr.Port()), uintptr(maxPeers), uintptr(maxFrame))
	if handle == 0 {
		return nil, fmt.Errorf("open SkyENet host on %s", addr)
	}
	host.handle = handle
	host.buffer = make([]byte, maxFrame)
	return host, nil
}

func (host *Native) Poll(timeout time.Duration) (Event, bool, error) {
	if host.handle == 0 {
		return Event{}, false, errors.New("SkyENet host is closed")
	}
	if timeout < 0 || timeout > time.Duration(^uint32(0))*time.Millisecond {
		return Event{}, false, errors.New("SkyENet poll timeout is out of range")
	}

	var received nativeEvent
	result, _, _ := host.poll.Call(
		host.handle, uintptr(timeout/time.Millisecond), uintptr(unsafe.Pointer(&received)),
		uintptr(unsafe.Pointer(unsafe.SliceData(host.buffer))), uintptr(len(host.buffer)),
	)
	switch kind := int32(result); kind {
	case 0:
		return Event{}, false, nil
	case int32(Connected), int32(Disconnected), int32(Received):
		if received.dataLength > uint64(len(host.buffer)) {
			return Event{}, false, errors.New("SkyENet bridge returned an invalid packet length")
		}
		return Event{
			Kind:     EventKind(kind),
			Peer:     received.peer,
			Channel:  uint8(received.channel),
			Reliable: received.flags&reliableFlag != 0,
			Data:     append([]byte(nil), host.buffer[:received.dataLength]...),
		}, true, nil
	default:
		return Event{}, false, fmt.Errorf("poll SkyENet host: native result %d", kind)
	}
}

func (host *Native) Send(peer uint64, channel uint8, reliable bool, data []byte) error {
	if host.handle == 0 {
		return errors.New("SkyENet host is closed")
	}
	flag := uintptr(0)
	if reliable {
		flag = 1
	}

	result, _, _ := host.send.Call(host.handle, uintptr(peer), uintptr(channel),
		uintptr(unsafe.Pointer(unsafe.SliceData(data))), uintptr(len(data)), flag)
	if int32(result) != 0 {
		return fmt.Errorf("send to SkyENet peer %d on channel %d", peer, channel)
	}
	return nil
}

func (host *Native) Disconnect(peer uint64, code uint32) {
	if host.handle != 0 {
		host.disconnect.Call(host.handle, uintptr(peer), uintptr(code))
	}
}

func (host *Native) Close() error {
	if host.handle == 0 {
		return nil
	}
	host.close.Call(host.handle)
	host.handle = 0
	return host.dll.Release()
}
