package lboverlay

import (
	"context"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
)

// ServeDNS implements the plugin.Plugin interface.
func (o *Overlay) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	// each should be updated every 10s, so older entries >1h could be removed.
	// check if normal query and do what is described in the README
	state := request.Request{W: w, Req: r}

	// handle health check update return reply
	if o.isHealthCheck(state) {
		for _, rr := range r.Extra {
			srv := rr.(*dns.SRV)
			o.setStatus(srv.Header().Name, srv.Port, status(srv.Header().Ttl))
		}
		resp := new(dns.Msg)
		resp.SetReply(r)
		w.WriteMsg(resp)
		return 0, nil
	}

	// Check if the qtype is A/AAAA/MX/SRV and do load balancing otherwise, just call the next plugin and call it a day.
	if state.QType() != dns.TypeA && state.QType() != dns.TypeAAAA && state.QType() != dns.TypeMX && state.QType() != dns.TypeSRV {
		return plugin.NextOrFailure(o.Name(), o.Next, ctx, w, r)
	}

	// responsewriter to do LB

	return 0, nil
}
