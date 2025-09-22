package rehosts

import (
	"context"
	"strings"
	"testing"

	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/coredns/coredns/plugin/test"

	"github.com/miekg/dns"
)

func TestLookupA(t *testing.T) {
	for _, tc := range hostsTestCases {
		m := tc.Msg()

		var tcFall fall.F
		isFall := tc.Qname == "fallthrough.owo."
		if isFall {
			tcFall = fall.Root
		} else {
			tcFall = fall.Zero
		}

		rh := Rehosts{
			Next: test.NextHandler(dns.RcodeNameError, nil),
			RehostsFile: &RehostsFile{
				Origins: []string{"."},
				records: nil,
				options: newOptions(),
			},
			Fall: tcFall,
		}
		rh.records = rh.parse(strings.NewReader(rehostsExample))

		rec := dnstest.NewRecorder(&test.ResponseWriter{})

		rcode, err := rh.ServeDNS(context.Background(), rec, m)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
			return
		}

		if isFall && tc.Rcode != rcode {
			t.Errorf("Expected rcode is %d, but got %d", tc.Rcode, rcode)
			return
		}

		if resp := rec.Msg; rec.Msg != nil {
			if err := test.SortAndCheck(resp, tc); err != nil {
				t.Error(err)
			}
		}
	}
}

var hostsTestCases = []test.Case{
	{
		Qname: "uwu.", Qtype: dns.TypeA,
		Answer: []dns.RR{
			test.A("uwu. 3600	IN	A 1.2.3.4"),
		},
	},
	{
		Qname: "uwu.", Qtype: dns.TypeA,
		Answer: []dns.RR{
			test.A("uwu. 3600	IN	A 1.2.3.4"),
		},
	},
	{
		Qname: "gato.", Qtype: dns.TypeAAAA,
		Answer: []dns.RR{
			test.AAAA("gato. 3600	IN	AAAA ::1"),
		},
	},
	{
		Qname: "uwu.", Qtype: dns.TypeAAAA,
		Answer: []dns.RR{},
	},
	{
		Qname: "uwu.", Qtype: dns.TypeMX,
		Answer: []dns.RR{},
	},
	{
		Qname: "fallthrough.owo.", Qtype: dns.TypeAAAA,
		Answer: []dns.RR{}, Rcode: dns.RcodeSuccess,
	},
}

const rehostsExample = `
127.0.0.1 gato
::1 gato my-gato gato.int
1.2.3.4 uwu
::FFFF:1.2.3.4 uwu
4.3.2.1 fallthrough.owo
`
