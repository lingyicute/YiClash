package shadowsocks

import (
	"net"

	"github.com/lingyicute/yiclashcore/adapter/inbound"
	N "github.com/lingyicute/yiclashcore/common/net"
	"github.com/lingyicute/yiclashcore/common/sockopt"
	C "github.com/lingyicute/yiclashcore/constant"
	"github.com/lingyicute/yiclashcore/log"
	"github.com/lingyicute/yiclashcore/transport/shadowsocks/core"
	"github.com/lingyicute/yiclashcore/transport/socks5"
)

type UDPListener struct {
	packetConn net.PacketConn
	closed     bool
}

func NewUDP(addr string, pickCipher core.Cipher, tunnel C.Tunnel) (*UDPListener, error) {
	l, err := net.ListenPacket("udp", addr)
	if err != nil {
		return nil, err
	}

	err = sockopt.UDPReuseaddr(l.(*net.UDPConn))
	if err != nil {
		log.Warnln("Failed to Reuse UDP Address: %s", err)
	}

	sl := &UDPListener{l, false}
	conn := pickCipher.PacketConn(N.NewEnhancePacketConn(l))
	go func() {
		for {
			data, put, remoteAddr, err := conn.WaitReadFrom()
			if err != nil {
				if put != nil {
					put()
				}
				if sl.closed {
					break
				}
				continue
			}
			handleSocksUDP(conn, tunnel, data, put, remoteAddr)
		}
	}()

	return sl, nil
}

func (l *UDPListener) Close() error {
	l.closed = true
	return l.packetConn.Close()
}

func (l *UDPListener) LocalAddr() net.Addr {
	return l.packetConn.LocalAddr()
}

func handleSocksUDP(pc net.PacketConn, tunnel C.Tunnel, buf []byte, put func(), addr net.Addr, additions ...inbound.Addition) {
	tgtAddr := socks5.SplitAddr(buf)
	if tgtAddr == nil {
		// Unresolved UDP packet, return buffer to the pool
		if put != nil {
			put()
		}
		return
	}
	target := tgtAddr
	payload := buf[len(tgtAddr):]

	packet := &packet{
		pc:      pc,
		rAddr:   addr,
		payload: payload,
		put:     put,
	}
	tunnel.HandleUDPPacket(inbound.NewPacket(target, packet, C.SHADOWSOCKS, additions...))
}
