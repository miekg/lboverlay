package lboverlay

import (
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
)

type status int

const (
	statusUnknown status = iota
	statusUnhealthy
	statusHealthy
)

// Overlay implement the plugin.Plugin interface and holds the health status.
type Overlay struct {
	health map[string]status // hostname + ":port" -> health status
	hcname string

	mu sync.RWMutex // protects health

	Next plugin.Handler
}

// Name implement the plugin.Plugin interface.
func (o *Overlay) Name() string { return "lboverlay" }

// New returns a initialized pointer to an Overlay.
func New(hcname string) *Overlay {
	if hcname == "" {
		hcname = "."
	}

	return &Overlay{health: make(map[string]status), hcname: dns.Fqdn(hcname)}
}

func (o *Overlay) setStatus(host string, port uint16, s status) {
	o.mu.Lock()
	o.health[joinHostPort(host, port)] = s
	o.mu.Unlock()
}

func (o *Overlay) status(host string, port uint16) status {
	o.mu.RLock()
	s, ok := o.health[joinHostPort(host, port)]
	o.mu.RUnlock()
	if ok {
		return s
	}
	return statusUnknown
}

func (o *Overlay) removeStatus(host string, port uint16) {
	o.mu.Lock()
	delete(o.health, joinHostPort(host, port))
	o.mu.Unlock()
}

func joinHostPort(host string, port uint16) string {
	return strings.ToLower(host) + ":" + strconv.Itoa(int(port))
}

func (o *Overlay) isHealthCheck(state request.Request) bool {
	if state.QName() != o.hcname {
		return false
	}
	if len(state.Req.Extra) == 0 {
		return false
	}
	// expect SRVs with owner root in the additional
	for _, rr := range state.Req.Extra {
		if _, ok := rr.(*dns.OPT); ok {
			continue
		}
		srv, ok := rr.(*dns.SRV)
		if !ok {
			return false
		}
		if srv.Header().Name != "." {
			return false
		}
		if srv.Header().Ttl > uint32(statusHealthy) {
			return false
		}
	}

	return true
}

// ResponseWriter is a response writer that takes loadbalancing into account
type ResponseWriter struct {
	dns.ResponseWriter
	*Overlay
}

func newResponseWriter(state request.Request, o *Overlay) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: state.W,
		Overlay:        o,
	}
}

// RemoteAddr implements the dns.ResponseWriter interface.
func (w *ResponseWriter) RemoteAddr() net.Addr {
	if w.remoteAddr != nil {
		return w.remoteAddr
	}
	return w.ResponseWriter.RemoteAddr()
}

// WriteMsg implements the dns.ResponseWriter interface.
func (w *ResponseWriter) WriteMsg(res *dns.Msg) error {
	// iterate over response, check
}
