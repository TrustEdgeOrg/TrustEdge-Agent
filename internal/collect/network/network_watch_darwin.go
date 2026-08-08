//go:build darwin

package network

import (
	"context"
	"syscall"
	"unsafe"
)

const (
	afRoute = 17

	rtmAdd     = 0x1
	rtmDelete  = 0x2
	rtmNewAddr = 0xc
	rtmDelAddr = 0xd
	rtmIfinfo  = 0xe
	rtmIfinfo2 = 0x12
)

func (m *NetworkMonitor) runPlatformWatcher(ctx context.Context, signal chan<- NetworkChangeReason) {
	if m.watchRouteSocket(ctx, signal) {
		return
	}
	m.logf("network watcher: route socket unavailable; link events disabled (heartbeat only)")
}

func (m *NetworkMonitor) watchRouteSocket(ctx context.Context, signal chan<- NetworkChangeReason) bool {
	fd, err := syscall.Socket(afRoute, syscall.SOCK_RAW, syscall.AF_UNSPEC)
	if err != nil {
		m.logf("network route socket: %v", err)
		return false
	}
	defer syscall.Close(fd)

	var addr syscall.RawSockaddrAny
	addr.Addr.Family = syscall.AF_ROUTE
	addr.Addr.Len = syscall.SizeofSockaddrAny
	_, _, errno := syscall.RawSyscall(
		syscall.SYS_BIND,
		uintptr(fd),
		uintptr(unsafe.Pointer(&addr)),
		uintptr(syscall.SizeofSockaddrAny),
	)
	if errno != 0 {
		m.logf("network route bind: %v", errno)
		return false
	}

	m.logf("network watcher: route socket active")

	buf := make([]byte, 2048)
	for {
		select {
		case <-ctx.Done():
			return true
		default:
		}

		n, err := syscall.Read(fd, buf)
		if err != nil {
			if ctx.Err() != nil {
				return true
			}
			m.logf("network route read: %v", err)
			return false
		}
		if n < 4 || !routeMsgInteresting(buf[3]) {
			continue
		}
		select {
		case signal <- NetworkReasonLink:
		default:
		}
	}
}

func routeMsgInteresting(msgType byte) bool {
	switch msgType {
	case rtmAdd, rtmDelete, rtmNewAddr, rtmDelAddr, rtmIfinfo, rtmIfinfo2:
		return true
	default:
		return false
	}
}
