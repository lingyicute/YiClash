package socks

import (
	"net"

	"github.com/lingyicute/yiclashcore/adapter/inbound"
	N "github.com/lingyicute/yiclashcore/common/net"
	"github.com/lingyicute/yiclashcore/common/sockopt"
	C "github.com/lingyicute/yiclashcore/constant"
	"github.com/lingyicute/yiclashcore/log"
	"github.com/lingyicute/yiclashcore/transport/socks5"
)

type UDPListener struct {
	packetConn net.PacketConn
	addr       string
	closed     bool
}

// RawAddress implements C.Listener
func (l *UDPListener) RawAddress() string {
	return l.addr
}

// Address implements C.Listener
func (l *UDPListener) Address() string {
	return l.packetConn.LocalAddr().String()
}

// Close implements C.Listener
func (l *UDPListener) Close() error {
	l.closed = true
	return l.packetConn.Close()
}

func NewUDP(addr string, tunnel C.Tunnel, additions ...inbound.Addition) (*UDPListener, error) {
	if len(additions) == 0 {
		additions = []inbound.Addition{
			inbound.WithInName("DEFAULT-SOCKS"),
			inbound.WithSpecialRules(""),
		}
	}
	l, err := net.ListenPacket("udp", addr)
	if err != nil {
		return nil, err
	}

	if err := sockopt.UDPReuseaddr(l.(*net.UDPConn)); err != nil {
		log.Warnln("Failed to Reuse UDP Address: %s", err)
	}

	sl := &UDPListener{
		packetConn: l,
		addr:       addr,
	}
	conn := N.NewEnhancePacketConn(l)
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
			handleSocksUDP(l, tunnel, data, put, remoteAddr, additions...)
		}
	}()

	return sl, nil
}

func handleSocksUDP(pc net.PacketConn, tunnel C.Tunnel, buf []byte, put func(), addr net.Addr, additions ...inbound.Addition) {
	target, payload, err := socks5.DecodeUDPPacket(buf)
	if err != nil {
		// Unresolved UDP packet, return buffer to the pool
		if put != nil {
			put()
		}
		return
	}
	packet := &packet{
		pc:      pc,
		rAddr:   addr,
		payload: payload,
		put:     put,
	}
	tunnel.HandleUDPPacket(inbound.NewPacket(target, packet, C.SOCKS5, additions...))
}
