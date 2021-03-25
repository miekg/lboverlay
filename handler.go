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
			srv, ok := rr.(*dns.SRV)
			if !ok {
				log.Debugf("Non SRV record in health check: %s", rr)
				continue
			}
			log.Debugf("Health status for %q set to: %s", joinHostPort(srv.Target, srv.Port), status(srv.Header().Ttl))
			o.setStatus(srv, status(srv.Header().Ttl))
		}
		resp := new(dns.Msg)
		resp.SetReply(r)
		w.WriteMsg(resp)
		return 0, nil
	}

	// We call the backend with SRV records, this means we can handle SRV records because that qeury comes round
	// to this plugin again, creating a cycle. It would be nice to handle SRV records as well (and overlay health to them) though.
	if state.QType() == dns.TypeSRV {
		return plugin.NextOrFailure(o.Name(), o.Next, ctx, w, r)
	}

	resp, err := o.u.Lookup(ctx, state, state.Name(), dns.TypeSRV)
	if err != nil {
		return plugin.NextOrFailure(o.Name(), o.Next, ctx, w, r)
	}
	// check the response beforehand to make code below simpler because less corner cases.
	srvs := 0
	for _, rr := range resp.Answer {
		if _, ok := rr.(*dns.SRV); ok {
			srvs++
		}
	}
	if srvs == 0 || len(resp.Answer) != srvs { // the response doesn't have srv in it, call NextOrFailure and be a noop
		return plugin.NextOrFailure(o.Name(), o.Next, ctx, w, r)
	}

	// check what we have, as we should have SRVs, but might not have all A/AAAA/MX records
	healthySRVs := make([]*dns.SRV, 0, len(resp.Answer))
	for _, rr := range resp.Answer {
		srv := rr.(*dns.SRV)
		s := o.status(srv)
		log.Debugf("Health status for %q is: %s", joinHostPort(srv.Target, srv.Port), s)
		if s == statusUnhealthy {
			continue
		}
		healthySRVs = append(healthySRVs, srv)
	}
	// for the healthy SRVs we need to resolve the target names with the original qtype from the query
	m := new(dns.Msg)
	m.SetReply(r)
	m.Answer = make([]dns.RR, 0, len(healthySRVs)) // there may be more rr than that returned though
	for _, srv := range healthySRVs {
		resp, err := o.u.Lookup(ctx, state, srv.Target, state.QType())
		if err != nil {
			continue
		}
		log.Debugf("Found answer for %s/%d, adding %d record(s)", srv.Target, state.QType(), len(resp.Answer))
		if len(resp.Answer) > 0 {
			m.Answer = append(m.Answer, resp.Answer...)
		}
	}
	// nodata, nxdomain and the like. TODO

	w.WriteMsg(m)

	return dns.RcodeSuccess, nil
}
