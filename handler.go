package lboverlay

import (
	"context"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metrics"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
)

// ServeDNS implements the plugin.Plugin interface.
func (o *Overlay) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	// each HC entity should be updated every 10s, so older entries >1h could be removed. TODO(miek)
	state := request.Request{W: w, Req: r}

	if o.isHealthCheck(state) {
		for _, rr := range r.Extra {
			srv, ok := rr.(*dns.SRV)
			if !ok {
				continue
			}
			log.Debugf("Health status for %q set to: %s", joinHostPort(srv.Target, srv.Port), status(srv.Header().Ttl))
			o.setStatus(srv, status(srv.Header().Ttl))
			hcCount.WithLabelValues(metrics.WithServer(ctx)).Inc()
		}
		resp := new(dns.Msg)
		resp.SetReply(r)
		w.WriteMsg(resp)
		return 0, nil
	}

	// if we see a SRV reply, we wrap it on our response writer and filter out the baddies in our own responseWriter
	if state.QType() == dns.TypeSRV {
		ow := &ResponseWriter{w, o}
		return plugin.NextOrFailure(o.Name(), o.Next, ctx, ow, r)
	}

	// By doing this lookup we get into the ow writer from above. This means we don't
	// have to check health anymore, because that is done the responseWriter already.
	resp, err := o.u.Lookup(ctx, state, state.Name(), dns.TypeSRV)
	if err != nil {
		log.Warning(err)
		return plugin.NextOrFailure(o.Name(), o.Next, ctx, w, r)
	}
	if x := resp.Rcode; x != dns.RcodeSuccess {
		resp.SetReply(r) // overwrites rcode
		resp.Rcode = x
		w.WriteMsg(resp)
		return 0, nil
	}

	// check the response beforehand to make code below simpler because less corner cases.
	srvs := 0
	for _, rr := range resp.Answer {
		if _, ok := rr.(*dns.SRV); ok {
			srvs++
		}
	}
	if srvs == 0 || len(resp.Answer) != srvs { // the response doesn't have (enough) SRV in it, call NextOrFailure and be a noop
		return plugin.NextOrFailure(o.Name(), o.Next, ctx, w, r)
	}

	m := new(dns.Msg)
	m.SetReply(r)
	rrs := make([]dns.RR, 0, len(resp.Answer))
	for _, s := range resp.Answer {
		srv := s.(*dns.SRV)
		// inspecting the additional section above might alleviate the extra queries here. TODO(miek)
		resp, err := o.u.Lookup(ctx, state, srv.Target, state.QType())
		if err != nil {
			continue
		}
		log.Debugf("Found %d records(1) for %s/%d", len(resp.Answer), srv.Target, state.QType())
		for _, rr := range resp.Answer {
			rr1 := dns.Copy(rr)
			rr1.Header().Name = state.QName()
			rr1.Header().Ttl = 5
			rrs = append(rrs, rr1)
		}
	}
	m.Answer = rrs
	w.WriteMsg(m)

	return dns.RcodeSuccess, nil
}
