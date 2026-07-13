//go:build linux

package collect

import (
	"context"
	"unsafe"

	"golang.org/x/sys/unix"
)

func (m *NetworkMonitor) runPlatformWatcher(ctx context.Context, signal chan<- NetworkChangeReason) {
	if m.watchNetlink(ctx, signal) {
		return
	}
	m.logf("network watcher: netlink unavailable; link events disabled (heartbeat only)")
}

func (m *NetworkMonitor) watchNetlink(ctx context.Context, signal chan<- NetworkChangeReason) bool {
	fd, err := unix.Socket(unix.AF_NETLINK, unix.SOCK_RAW, unix.NETLINK_ROUTE)
	if err != nil {
		m.logf("network netlink socket: %v", err)
		return false
	}
	defer unix.Close(fd)

	addr := &unix.SockaddrNetlink{Family: unix.AF_NETLINK}
	if err := unix.Bind(fd, addr); err != nil {
		m.logf("network netlink bind: %v", err)
		return false
	}

	m.logf("network watcher: netlink route socket active")

	buf := make([]byte, 8192)
	for {
		select {
		case <-ctx.Done():
			return true
		default:
		}

		n, err := unix.Read(fd, buf)
		if err != nil {
			if ctx.Err() != nil {
				return true
			}
			m.logf("network netlink read: %v", err)
			return false
		}
		if n < unix.SizeofNlMsghdr {
			continue
		}
		for i := 0; i < n; {
			if i+unix.SizeofNlMsghdr > n {
				break
			}
			hdr := (*unix.NlMsghdr)(unsafe.Pointer(&buf[i]))
			if hdr.Len < unix.SizeofNlMsghdr {
				break
			}
			if netlinkMsgInteresting(hdr.Type) {
				select {
				case signal <- NetworkReasonLink:
				default:
				}
				break
			}
			if hdr.Len == 0 {
				break
			}
			i += int(hdr.Len)
		}
	}
}

func netlinkMsgInteresting(msgType uint16) bool {
	switch msgType {
	case unix.RTM_NEWLINK, unix.RTM_DELLINK, unix.RTM_NEWADDR, unix.RTM_DELADDR:
		return true
	default:
		return false
	}
}
