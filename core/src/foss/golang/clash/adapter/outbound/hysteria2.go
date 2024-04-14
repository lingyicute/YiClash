package outbound

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"runtime"
	"strconv"
	"time"

	CN "github.com/lingyicute/yiclashcore/common/net"
	"github.com/lingyicute/yiclashcore/common/utils"
	"github.com/lingyicute/yiclashcore/component/ca"
	"github.com/lingyicute/yiclashcore/component/dialer"
	"github.com/lingyicute/yiclashcore/component/proxydialer"
	C "github.com/lingyicute/yiclashcore/constant"
	"github.com/lingyicute/yiclashcore/log"
	tuicCommon "github.com/lingyicute/yiclashcore/transport/tuic/common"

	"github.com/metacubex/sing-quic/hysteria2"

	M "github.com/sagernet/sing/common/metadata"
	"github.com/zhangyunhao116/fastrand"
)

func init() {
	hysteria2.SetCongestionController = tuicCommon.SetCongestionController
}

const minHopInterval = 5
const defaultHopInterval = 30

type Hysteria2 struct {
	*Base

	option *Hysteria2Option
	client *hysteria2.Client
	dialer proxydialer.SingDialer
}

type Hysteria2Option struct {
	BasicOption
	Name           string   `proxy:"name"`
	Server         string   `proxy:"server"`
	Port           int      `proxy:"port,omitempty"`
	Ports          string   `proxy:"ports,omitempty"`
	HopInterval    int      `proxy:"hop-interval,omitempty"`
	Up             string   `proxy:"up,omitempty"`
	Down           string   `proxy:"down,omitempty"`
	Password       string   `proxy:"password,omitempty"`
	Obfs           string   `proxy:"obfs,omitempty"`
	ObfsPassword   string   `proxy:"obfs-password,omitempty"`
	SNI            string   `proxy:"sni,omitempty"`
	SkipCertVerify bool     `proxy:"skip-cert-verify,omitempty"`
	Fingerprint    string   `proxy:"fingerprint,omitempty"`
	ALPN           []string `proxy:"alpn,omitempty"`
	CustomCA       string   `proxy:"ca,omitempty"`
	CustomCAString string   `proxy:"ca-str,omitempty"`
	CWND           int      `proxy:"cwnd,omitempty"`
	UdpMTU         int      `proxy:"udp-mtu,omitempty"`
}

func (h *Hysteria2) DialContext(ctx context.Context, metadata *C.Metadata, opts ...dialer.Option) (_ C.Conn, err error) {
	options := h.Base.DialOptions(opts...)
	h.dialer.SetDialer(dialer.NewDialer(options...))
	c, err := h.client.DialConn(ctx, M.ParseSocksaddrHostPort(metadata.String(), metadata.DstPort))
	if err != nil {
		return nil, err
	}
	return NewConn(CN.NewRefConn(c, h), h), nil
}

func (h *Hysteria2) ListenPacketContext(ctx context.Context, metadata *C.Metadata, opts ...dialer.Option) (_ C.PacketConn, err error) {
	options := h.Base.DialOptions(opts...)
	h.dialer.SetDialer(dialer.NewDialer(options...))
	pc, err := h.client.ListenPacket(ctx)
	if err != nil {
		return nil, err
	}
	if pc == nil {
		return nil, errors.New("packetConn is nil")
	}
	return newPacketConn(CN.NewRefPacketConn(CN.NewThreadSafePacketConn(pc), h), h), nil
}

func closeHysteria2(h *Hysteria2) {
	if h.client != nil {
		_ = h.client.CloseWithError(errors.New("proxy removed"))
	}
}

func NewHysteria2(option Hysteria2Option) (*Hysteria2, error) {
	addr := net.JoinHostPort(option.Server, strconv.Itoa(option.Port))
	var salamanderPassword string
	if len(option.Obfs) > 0 {
		if option.ObfsPassword == "" {
			return nil, errors.New("missing obfs password")
		}
		switch option.Obfs {
		case hysteria2.ObfsTypeSalamander:
			salamanderPassword = option.ObfsPassword
		default:
			return nil, fmt.Errorf("unknown obfs type: %s", option.Obfs)
		}
	}

	serverName := option.Server
	if option.SNI != "" {
		serverName = option.SNI
	}

	tlsConfig := &tls.Config{
		ServerName:         serverName,
		InsecureSkipVerify: option.SkipCertVerify,
		MinVersion:         tls.VersionTLS13,
	}

	var err error
	tlsConfig, err = ca.GetTLSConfig(tlsConfig, option.Fingerprint, option.CustomCA, option.CustomCAString)
	if err != nil {
		return nil, err
	}

	if len(option.ALPN) > 0 {
		tlsConfig.NextProtos = option.ALPN
	}

	if option.UdpMTU == 0 {
		// "1200" from quic-go's MaxDatagramSize
		// "-3" from quic-go's DatagramFrame.MaxDataLen
		option.UdpMTU = 1200 - 3
	}

	singDialer := proxydialer.NewByNameSingDialer(option.DialerProxy, dialer.NewDialer())

	clientOptions := hysteria2.ClientOptions{
		Context:            context.TODO(),
		Dialer:             singDialer,
		Logger:             log.SingLogger,
		SendBPS:            StringToBps(option.Up),
		ReceiveBPS:         StringToBps(option.Down),
		SalamanderPassword: salamanderPassword,
		Password:           option.Password,
		TLSConfig:          tlsConfig,
		UDPDisabled:        false,
		CWND:               option.CWND,
		UdpMTU:             option.UdpMTU,
		ServerAddress: func(ctx context.Context) (*net.UDPAddr, error) {
			return resolveUDPAddrWithPrefer(ctx, "udp", addr, C.NewDNSPrefer(option.IPVersion))
		},
	}

	var ranges utils.IntRanges[uint16]
	var serverAddress []string
	if option.Ports != "" {
		ranges, err = utils.NewUnsignedRanges[uint16](option.Ports)
		if err != nil {
			return nil, err
		}
		ranges.Range(func(port uint16) bool {
			serverAddress = append(serverAddress, net.JoinHostPort(option.Server, strconv.Itoa(int(port))))
			return true
		})
		if len(serverAddress) > 0 {
			clientOptions.ServerAddress = func(ctx context.Context) (*net.UDPAddr, error) {
				return resolveUDPAddrWithPrefer(ctx, "udp", serverAddress[fastrand.Intn(len(serverAddress))], C.NewDNSPrefer(option.IPVersion))
			}

			if option.HopInterval == 0 {
				option.HopInterval = defaultHopInterval
			} else if option.HopInterval < minHopInterval {
				option.HopInterval = minHopInterval
			}
			clientOptions.HopInterval = time.Duration(option.HopInterval) * time.Second
		}
	}
	if option.Port == 0 && len(serverAddress) == 0 {
		return nil, errors.New("invalid port")
	}

	client, err := hysteria2.NewClient(clientOptions)
	if err != nil {
		return nil, err
	}

	outbound := &Hysteria2{
		Base: &Base{
			name:   option.Name,
			addr:   addr,
			tp:     C.Hysteria2,
			udp:    true,
			iface:  option.Interface,
			rmark:  option.RoutingMark,
			prefer: C.NewDNSPrefer(option.IPVersion),
		},
		option: &option,
		client: client,
		dialer: singDialer,
	}
	runtime.SetFinalizer(outbound, closeHysteria2)

	return outbound, nil
}
