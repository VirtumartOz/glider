package tcp

import (
	"errors"
	"net"
	"net/url"
	"strings"

	"github.com/nadoo/glider/log"
	"github.com/nadoo/glider/proxy"
)

// TCP struct.
type TCP struct {
	dialer proxy.Dialer
	proxy  proxy.Proxy
	server proxy.Server
	addr   string
}

func init() {
	proxy.RegisterDialer("tcp", NewTCPDialer)
	proxy.RegisterServer("tcp", NewTCPServer)
}

// NewTCP returns a tcp struct.
func NewTCP(s string, d proxy.Dialer, p proxy.Proxy) (*TCP, error) {
	u, err := url.Parse(s)
	if err != nil {
		log.F("[tls] parse url err: %s", err)
		return nil, err
	}

	t := &TCP{
		dialer: d,
		proxy:  p,
		addr:   u.Host,
	}

	return t, nil
}

// NewTCPDialer returns a tcp dialer.
func NewTCPDialer(s string, d proxy.Dialer) (proxy.Dialer, error) {
	return NewTCP(s, d, nil)
}

// NewTCPServer returns a tcp transport layer before the real server.
func NewTCPServer(s string, p proxy.Proxy) (proxy.Server, error) {
	// transport := strings.Split(s, ",")

	// prepare transport listener
	// TODO: check here
	// if len(transport) < 2 {
	// 	return nil, errors.New("[tcp] malformd listener:" + s)
	// }

	// t.server, err = proxy.ServerFromURL(transport[1], p)
	// if err != nil {
	// 	return nil, err
	// }

	return NewTCP(s, nil, p)
}

// ListenAndServe listens on server's addr and serves connections.
func (s *TCP) ListenAndServe() {
	l, err := net.Listen("tcp", s.addr)
	if err != nil {
		log.F("[tcp] failed to listen on %s: %v", s.addr, err)
		return
	}
	defer l.Close()

	log.F("[tcp] listening TCP on %s", s.addr)

	for {
		c, err := l.Accept()
		if err != nil {
			log.F("[tcp] failed to accept: %v", err)
			continue
		}

		go s.Serve(c)
	}
}

// Serve serves a connection.
func (s *TCP) Serve(c net.Conn) {
	// we know the internal server will close the connection after serve
	// defer c.Close()

	if s.server != nil {
		s.server.Serve(c)
		return
	}

	defer c.Close()

	if c, ok := c.(*net.TCPConn); ok {
		c.SetKeepAlive(true)
	}

	rc, dialer, err := s.proxy.Dial("tcp", "")
	if err != nil {
		log.F("[tcp] %s <-> %s via %s, error in dial: %v", c.RemoteAddr(), s.addr, dialer.Addr(), err)
		s.proxy.Record(dialer, false)
		return
	}
	defer rc.Close()

	log.F("[tcp] %s <-> %s", c.RemoteAddr(), dialer.Addr())

	if err = proxy.Relay(c, rc); err != nil {
		log.F("[tcp] %s <-> %s, relay error: %v", c.RemoteAddr(), dialer.Addr(), err)
		// record remote conn failure only
		if !strings.Contains(err.Error(), s.addr) {
			s.proxy.Record(dialer, false)
		}
	}
}

// Addr returns forwarder's address.
func (s *TCP) Addr() string {
	if s.addr == "" {
		return s.dialer.Addr()
	}
	return s.addr
}

// Dial connects to the address addr on the network net via the proxy.
func (s *TCP) Dial(network, addr string) (net.Conn, error) {
	return s.dialer.Dial("tcp", s.addr)
}

// DialUDP connects to the given address via the proxy.
func (s *TCP) DialUDP(network, addr string) (net.PacketConn, net.Addr, error) {
	return nil, nil, errors.New("tcp client does not support udp now")
}
