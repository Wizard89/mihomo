package gun

import (
	"net"
	"net/http"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/net/http2"
)

type clientConnPool struct {
	t     *http2.Transport
	mu    sync.Mutex
	conns map[string][]*http2.ClientConn // key is host:port
}

type clientConn struct {
	t     *http.Transport
	tconn net.Conn // usually *tls.Conn, except specialized impls
}

type efaceWords struct {
	typ  unsafe.Pointer
	data unsafe.Pointer
}

type tlsConn interface {
	net.Conn
	NetConn() net.Conn
}

func closeClientConn(cc *http2.ClientConn) { // like forceCloseConn() in http2.ClientConn but also apply for tls-like conn
	if conn, ok := (*clientConn)(unsafe.Pointer(cc)).tconn.(tlsConn); ok {
		t := time.AfterFunc(time.Second, func() {
			_ = conn.NetConn().Close()
		})
		defer t.Stop()
	}
	_ = cc.Close()
}

func (tw *TransportWrap) Close() error {
	connPool := transportConnPool(tw.Transport)
	p := (*clientConnPool)((*efaceWords)(unsafe.Pointer(&connPool)).data)
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, vv := range p.conns {
		for _, cc := range vv {
			closeClientConn(cc)
		}
	}
	return nil
}

//go:linkname transportConnPool golang.org/x/net/http2.(*Transport).connPool
func transportConnPool(t *http2.Transport) http2.ClientConnPool
