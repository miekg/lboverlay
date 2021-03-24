package lboverlay

import (
	"net"
	"sync"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/upstream"

	"github.com/miekg/dns"
)

type status int

const (
	statusUnknown status = iota
	statusUnhealthy
	statusHealthy
)

// Overlay implement the plugin.Plugin interface and holds the health status
type Overlay struct {
	health map[string]status // hostname + ":port" -> health status
	hcname string

	mu sync.RWMutex       // protects health
	u  *upstream.Upstream // used to query the backend

	Next plugin.Handler
}

func (o *Overlay) Name() string { return "lboverlay" }

func New(hcname string) *Overlay {
	if hcname == "" {
		hcname = "."
	}

	return &Overlay{health: make(map[string]status), u: new(upstream.Upstream), hcname: dns.Fqdn(hcname)}
}

func (o *Overlay) SetStatus(host, port string, s status) {
	o.mu.Lock()
	o.health[net.JoinHostPort(host, port)] = s
	o.mu.Unlock()
}

func (o *Overlay) Status(host, port string) status {
	o.mu.RLock()
	s, ok := o.health[net.JoinHostPort(host, port)]
	o.mu.RUnlock()
	if ok {
		return s
	}
	return statusUnknown

}

func (o *Overlay) RemoveStatus(host, port string) {
	o.mu.Lock()
	delete(o.health, net.JoinHostPort(host, port))
	o.mu.Unlock()
}
