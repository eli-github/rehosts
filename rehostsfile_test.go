// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rehosts

import (
	"net"
	"strings"
	"testing"

	"github.com/coredns/coredns/plugin"
)

func newRehostsFile(file string) *RehostsFile {
	rh := &RehostsFile{
		Origins: []string{"."},
		records: nil,
		options: newOptions(),
	}
	rh.records = rh.parse(strings.NewReader(file))
	return rh
}

type staticHostEntry struct {
	in string
	v4 []string
	v6 []string
}

var (
	hosts = `
	#regular
	127.0.0.1 uwu aoa
	1234::cDEf owo
	127.0.0.3 ouo

	# wildcard
	127.0.1.1 *.owo.uwu
	127.0.1.2 *.uwu

	# regexp
	127.0.2.1 @ go+gle\.com?
	127.0.2.2 @ (porn|git)hub.com
	`
	overrideHosts = `
	127.0.0.1 google.com t-google.com *.my-google.us
	127.0.0.2 @ .*not-google\.com
	127.0.0.3 *google.com

	127.0.1.1 google.com t-google.com *.my-google.us
	127.0.1.2 @ .*not-google\.com
	127.0.1.3 *google.com
	`
	singleLineHosts = `127.0.0.1     gato`
	ipv4Hosts       = `
	127.0.0.1 owo
	`
	ipv6Hosts = `
	BEba::1234 uwu
	`
)

var lookupStaticHostTests = []struct {
	file string
	ents []staticHostEntry
}{
	{
		hosts,
		[]staticHostEntry{
			{"rawr.", []string{}, []string{}},
			{"uwu.", []string{"127.0.0.1"}, []string{}},
			{"aoa.", []string{"127.0.0.1"}, []string{}},
			{"owo.", []string{}, []string{"1234::cdef"}},
			{"ouo.", []string{"127.0.0.3"}, []string{}},

			{"ucu.ouo.owo.uwu.", []string{"127.0.1.1"}, []string{}},
			{"ouo.owo.uwu.", []string{"127.0.1.1"}, []string{}},
			{"aoa.ouo.uwu.", []string{"127.0.1.2"}, []string{}},
			{"ouo.uwu.", []string{"127.0.1.2"}, []string{}},

			{"gogle.com.", []string{"127.0.2.1"}, []string{}},
			{"gogle.co.", []string{"127.0.2.1"}, []string{}},
			{"gooooooooooooooooooooooooooooogle.co.", []string{"127.0.2.1"}, []string{}},
			{"github.com.", []string{"127.0.2.2"}, []string{}},
			{"pornhub.com.", []string{"127.0.2.2"}, []string{}},
		},
	},
	{
		overrideHosts,
		[]staticHostEntry{
			{"gle.com.", []string{}, []string{}},
			{"google.com.", []string{"127.0.0.1"}, []string{}},
			{"t-google.com.", []string{"127.0.0.1"}, []string{}},
			{"not.my-google.us.", []string{"127.0.0.1"}, []string{}},
			{"why-not-google.com.", []string{"127.0.0.2"}, []string{}},
			{"why-google.com.", []string{"127.0.0.3"}, []string{}},
			{"not-google.com.", []string{"127.0.0.2"}, []string{}},
		},
	},
	{
		singleLineHosts,
		[]staticHostEntry{
			{"gato.", []string{"127.0.0.1"}, []string{}},
		},
	},
}

func TestLookupHosts(t *testing.T) {
	for _, tt := range lookupStaticHostTests {
		h := newRehostsFile(tt.file)
		for _, ent := range tt.ents {
			testHostsCases(t, ent, h)
		}
	}
}

func testHostsCases(t *testing.T, ent staticHostEntry, rh *RehostsFile) {
	ins := []string{ent.in, plugin.Name(ent.in).Normalize(), strings.ToLower(ent.in), strings.ToUpper(ent.in)}
	for k, in := range ins {
		addrsV4 := rh.LookupStaticHostV4(in)
		if len(addrsV4) != len(ent.v4) {
			t.Fatalf("%d, LookupStaticHostV4(%s) = %v; want %v", k, in, addrsV4, ent.v4)
		}
		for i, v4 := range addrsV4 {
			if v4.String() != ent.v4[i] {
				t.Fatalf("%d, LookupStaticHostV4(%s) = %v; want %v", k, in, addrsV4, ent.v4)
			}
		}
		addrsV6 := rh.LookupStaticHostV6(in)
		if len(addrsV6) != len(ent.v6) {
			t.Fatalf("%d, LookupStaticHostV6(%s) = %v; want %v", k, in, addrsV6, ent.v6)
		}
		for i, v6 := range addrsV6 {
			if v6.String() != ent.v6[i] {
				t.Fatalf("%d, LookupStaticHostV6(%s) = %v; want %v", k, in, addrsV6, ent.v6)
			}
		}
	}
}

func TestHostCacheModification(t *testing.T) {
	// Ensure that programs can't modify the internals of the host cache.
	// See https://github.com/golang/go/issues/14212.

	rh := newRehostsFile(ipv4Hosts)
	entry := staticHostEntry{"owo.", []string{"127.0.0.1"}, []string{}}
	testHostsCases(t, entry, rh)

	// Modify returned address
	addrs := rh.LookupStaticHostV4(entry.in)
	for i := range addrs {
		addrs[i] = net.IPv4zero
	}
	testHostsCases(t, entry, rh)

	rh = newRehostsFile(ipv6Hosts)
	entry = staticHostEntry{"uwu.", []string{}, []string{"beba::1234"}}
	testHostsCases(t, entry, rh)

	// Modify returned address
	addrs = rh.LookupStaticHostV6(entry.in)
	for i := range addrs {
		addrs[i] = net.IPv6zero
	}
	testHostsCases(t, entry, rh)
}
