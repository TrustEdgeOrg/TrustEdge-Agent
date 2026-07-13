//go:build linux

package collect

import (
	"context"
	"encoding/binary"
	"log"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
	"golang.org/x/sys/unix"
)

const (
	cnIdxProc            = 0x1
	cnValProc            = 0x1
	procCnMcastListen    = 1
	procEventExec        = 0x00000002
	procEventExit        = 0x80000000
	procConnectorNLGroup = 0x1
)

type linuxProcessWatcher struct {
	log *log.Logger
}

func newPlatformProcessWatcher(logger *log.Logger) ProcessWatcher {
	return &linuxProcessWatcher{log: logger}
}

func (w *linuxProcessWatcher) Run(ctx context.Context) <-chan ProcessChange {
	out := make(chan ProcessChange, 64)
	go func() {
		defer close(out)
		if err := w.run(ctx, out); err != nil && ctx.Err() == nil {
			w.logf("process watcher: %v (poll reconciliation continues)", err)
		}
	}()
	return out
}

func (w *linuxProcessWatcher) run(ctx context.Context, out chan<- ProcessChange) error {
	fd, err := unix.Socket(unix.AF_NETLINK, unix.SOCK_DGRAM, unix.NETLINK_CONNECTOR)
	if err != nil {
		return err
	}
	defer unix.Close(fd)

	addr := &unix.SockaddrNetlink{Family: unix.AF_NETLINK, Groups: procConnectorNLGroup}
	if err := unix.Bind(fd, addr); err != nil {
		return err
	}
	if err := sendProcListen(fd); err != nil {
		return err
	}
	w.logf("process watcher: netlink connector active")

	buf := make([]byte, 8192)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := unix.Read(fd, buf)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		for _, ch := range parseProcMessages(buf[:n]) {
			select {
			case out <- ch:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

func sendProcListen(fd int) error {
	cn := make([]byte, 20)
	binary.LittleEndian.PutUint32(cn[0:], cnIdxProc)
	binary.LittleEndian.PutUint32(cn[4:], cnValProc)
	binary.LittleEndian.PutUint16(cn[14:], 4)
	binary.LittleEndian.PutUint32(cn[16:], procCnMcastListen)

	nlh := make([]byte, unix.NLMSG_HDRLEN+len(cn))
	binary.LittleEndian.PutUint32(nlh[0:], uint32(unix.NLMSG_HDRLEN+len(cn)))
	binary.LittleEndian.PutUint16(nlh[4:], unix.NLMSG_DONE)
	binary.LittleEndian.PutUint16(nlh[6:], unix.NLM_F_REQUEST)
	copy(nlh[unix.NLMSG_HDRLEN:], cn)

	addr := &unix.SockaddrNetlink{Family: unix.AF_NETLINK}
	return unix.Sendto(fd, nlh, 0, addr)
}

func parseProcMessages(buf []byte) []ProcessChange {
	var out []ProcessChange
	for len(buf) >= unix.NLMSG_HDRLEN {
		l := int(binary.LittleEndian.Uint32(buf[0:4]))
		if l < unix.NLMSG_HDRLEN {
			break
		}
		if l > len(buf) {
			break
		}
		msg := buf[:l]
		buf = buf[l:]

		payload := msg[unix.NLMSG_HDRLEN:]
		if len(payload) < 20 {
			continue
		}
		dataLen := int(binary.LittleEndian.Uint16(payload[14:16]))
		if dataLen < 16 || len(payload) < 16+dataLen {
			continue
		}
		ev := payload[16 : 16+dataLen]
		if ch, ok := procEventChange(ev); ok {
			out = append(out, ch)
		}
	}
	return out
}

func procEventChange(ev []byte) (ProcessChange, bool) {
	if len(ev) < 16 {
		return ProcessChange{}, false
	}
	what := binary.LittleEndian.Uint32(ev[0:4])
	switch what {
	case procEventExec:
		pid := int(int32(binary.LittleEndian.Uint32(ev[16:20])))
		row, ok := processRowFromPID(pid)
		if !ok {
			row = processRow{PID: pid}
		}
		return ProcessChange{Type: constants.TypeProcessStart, Payload: processPayload(row)}, true
	case procEventExit:
		pid := int(int32(binary.LittleEndian.Uint32(ev[16:20])))
		ppid := int(int32(binary.LittleEndian.Uint32(ev[28:32])))
		row, ok := processRowFromPID(pid)
		if !ok {
			row = processRow{PID: pid, PPID: ppid}
		}
		return ProcessChange{
			Type: constants.TypeProcessExit,
			Payload: map[string]any{
				"pid":        row.PID,
				"ppid":       row.PPID,
				"user":       row.User,
				"comm":       row.Comm,
				"executable": row.Executable,
			},
		}, true
	default:
		return ProcessChange{}, false
	}
}

func (w *linuxProcessWatcher) logf(format string, args ...any) {
	if w.log != nil {
		w.log.Printf(format, args...)
	}
}
