package lboverlay

import (
	"strconv"
	"strings"
	"sync"

	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/plugin/pkg/upstream"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
)

var log = clog.NewWithPlugin("overlay")

type status int

const (
	statusUnknown status = iota
	statusUnhealthy
	statusHealthy
)

func (s status) String() string {
	switch s {
	default:
		fallthrough
	case statusUnknown:
		return "UNKNOWN"
	case statusUnhealthy:
		return "UNHEALTHY"
	case statusHealthy:
		return "HEALTHY"
	}
}

// Overlay implement the plugin.Plugin interface and holds the health status.
type Overlay struct {
	health map[string]status // hostname + ":port" -> health status
	hcname string
	u      *upstream.Upstream

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

	return &Overlay{health: make(map[string]status), hcname: dns.Fqdn(hcname), u: upstream.New()}
}

func (o *Overlay) setStatus(srv *dns.SRV, s status) {
	o.mu.Lock()
	o.health[joinHostPort(srv.Target, srv.Port)] = s
	o.mu.Unlock()
}

func (o *Overlay) status(srv *dns.SRV) status {
	o.mu.RLock()
	s, ok := o.health[joinHostPort(srv.Target, srv.Port)]
	o.mu.RUnlock()
	if ok {
		return s
	}
	return statusUnknown
}

func (o *Overlay) removeStatus(srv *dns.SRV) {
	o.mu.Lock()
	delete(o.health, joinHostPort(srv.Target, srv.Port))
	o.mu.Unlock()
}

func joinHostPort(host string, port uint16) string {
	return strings.ToLower(host) + ":" + strconv.Itoa(int(port))
}

func (o *Overlay) isHealthCheck(state request.Request) bool {
	if state.QName() != o.hcname {
		return false
	}
	if state.Qtype != dns.TypeHINFO {
		return false
	}
	if len(state.Req.Extra) == 0 {
		return false
	}
	if len(state.Req.Ns) != 0 {
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

// WriteMsg implements the dns.ResponseWriter interface.
func (w *ResponseWriter) WriteMsg(res *dns.Msg) error {
	if res.Rcode != dns.RcodeSuccess {
		w.ResponseWriter.WriteMsg(res)
		return nil
	}

	healthySRVs := make([]dns.RR, 0, len(res.Answer))
	for _, rr := range res.Answer {
		srv, ok := rr.(*dns.SRV)
		if !ok {
			continue
		}
		s := w.Overlay.status(srv)
		log.Debugf("Health status for %q is: %s", joinHostPort(srv.Target, srv.Port), s)
		if s == statusUnhealthy {
			continue
		}
		healthySRVs = append(healthySRVs, srv)
	}
	if len(healthySRVs) == len(res.Answer) {
		// don't modify packet, send as-is
		w.ResponseWriter.WriteMsg(res)
		return nil
	}
	// make new msg and send that
	m := new(dns.Msg)
	m.SetReply(res)
	m.Answer = make([]dns.RR, len(healthySRVs))
	for i, s := range healthySRVs {
		m.Answer[i] = dns.Copy(s)
		m.Answer[i].Header().Ttl = 5
	}
	m.Ns = res.Ns
	m.Extra = res.Extra
	w.ResponseWriter.WriteMsg(m)
	return nil
}
