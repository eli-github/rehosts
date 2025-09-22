package rehosts

import (
	"context"
	"net"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
)

type Rehosts struct {
	Next plugin.Handler
	*RehostsFile

	Fall fall.F
}

func (rh Rehosts) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	qname := state.Name()

	answers := []dns.RR{}

	zone := plugin.Zones(rh.Origins).Matches(qname)
	if zone == "" {
		// PTR zones don't need to be specified in Origins.
		if state.QType() != dns.TypePTR {
			// if this doesn't match we need to fall through regardless of h.Fallthrough
			return plugin.NextOrFailure(rh.Name(), rh.Next, ctx, w, r)
		}
	}

	switch state.QType() {
	case dns.TypeA:
		ips := rh.LookupStaticHostV4(qname)
		answers = a(qname, rh.options.ttl, ips)
	case dns.TypeAAAA:
		ips := rh.LookupStaticHostV6(qname)
		answers = aaaa(qname, rh.options.ttl, ips)
	}

	// Only on NXDOMAIN we will fallthrough.
	if len(answers) == 0 && !rh.otherRecordsExist(qname) {
		if rh.Fall.Through(qname) {
			return plugin.NextOrFailure(rh.Name(), rh.Next, ctx, w, r)
		}

		// We want to send an NXDOMAIN, but because of /etc/hosts' setup we don't have a SOA, so we make it SERVFAIL
		// to at least give an answer back to signals we're having problems resolving this.
		return dns.RcodeServerFailure, nil
	}

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	m.Answer = answers

	w.WriteMsg(m)
	return dns.RcodeSuccess, nil
}

func (h Rehosts) Name() string { return "rehosts" }

func (rh Rehosts) otherRecordsExist(qname string) bool {
	if len(rh.LookupStaticHostV4(qname)) > 0 {
		return true
	}
	if len(rh.LookupStaticHostV6(qname)) > 0 {
		return true
	}
	return false
}

// (from: coredns/plugins/hosts): a takes a slice of net.IPs and returns a slice of A RRs.
func a(zone string, ttl uint32, ips []net.IP) []dns.RR {
	answers := make([]dns.RR, len(ips))
	for i, ip := range ips {
		r := new(dns.A)
		r.Hdr = dns.RR_Header{Name: zone, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: ttl}
		r.A = ip
		answers[i] = r
	}
	return answers
}

// (from: coredns/plugins/hosts): aaaa takes a slice of net.IPs and returns a slice of AAAA RRs.
func aaaa(zone string, ttl uint32, ips []net.IP) []dns.RR {
	answers := make([]dns.RR, len(ips))
	for i, ip := range ips {
		r := new(dns.AAAA)
		r.Hdr = dns.RR_Header{Name: zone, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: ttl}
		r.AAAA = ip
		answers[i] = r
	}
	return answers
}
