package diff_tests

import (
	"context"
	"fmt"
	"io"
	"net/netip"
	"sync/atomic"
	"testing"
	"time"

	"codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
	"codeberg.org/miekg/dns/rdata"
	assert "github.com/stretchr/testify/assert"
)

var upstreamDNSserver = []string{
	"223.5.5.5",
	"223.6.6.6",
	"119.29.29.29",
	"2400:3200::1",
}

func QueryDNSRecord(
	domain, server string,
	dnsType uint16,
) []string {
	m := dns.NewMsg(domain, dnsType)
	m.ID = dns.ID()
	c := new(dns.Client)
	resp, _, err := c.Exchange(context.TODO(), m, "udp", server+":53")
	if err != nil {
		fmt.Printf("error: %v", err)
		return nil
	}
	if resp.Rcode != dns.RcodeSuccess {
		fmt.Printf("query error: %s", dns.RcodeToString[resp.Rcode])
		return nil
	}
	var res []string
	for _, ans := range resp.Answer {
		if a, ok := ans.(*dns.A); ok {
			res = append(res, a.String())
		} else if aaaa, ok := ans.(*dns.AAAA); ok {
			res = append(res, aaaa.String())
		} else {

		}
	}
	return res
}

// if fail to iterate, then use this function
func forwardToUpstream(req *dns.Msg) (*dns.Msg, error) {
	as_client := dns.NewClient()
	as_client.Dialer.Timeout = 5 * time.Second
	as_client.ReadTimeout = 5 * time.Second
	as_client.WriteTimeout = 5 * time.Second
	for _, stream := range upstreamDNSserver {
		resp, _, err := as_client.Exchange(context.TODO(), req, "tcp", stream)
		if err != nil {
			continue
		}

		return resp, nil
	}
	return nil, fmt.Errorf(
		"DNS-error-code:%d",
		dns.ExtendedErrorSynthesized,
	)
}

// TODO: as an honeypot, do we really need to maintain a DNS record?
func handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	msg := r.Copy()
	dnsutil.SetReply(msg, r)
	msg.Authoritative = true
	for _, q := range r.Question {
		qName := dnsutil.Canonical(q.Header().Name)
		h := dns.Header{Name: qName, Class: dns.ClassINET, TTL: 1800}
		switch dns.RRToType(q) {
		case dns.TypeA:
			rr := &dns.A{
				Hdr: h,
				A:   rdata.A{Addr: netip.Addr{}},
			}
			msg.Answer = append(msg.Answer, rr)
		case dns.TypeAAAA:
			rr := &dns.AAAA{
				Hdr:  h,
				AAAA: rdata.AAAA{Addr: netip.Addr{}},
			}
			msg.Answer = append(msg.Answer, rr)
		case dns.TypeCNAME:
			rr := &dns.CNAME{}
			msg.Answer = append(msg.Answer, rr)
		case dns.TypeTXT:
			rr := &dns.TXT{
				Hdr: h,
				TXT: rdata.TXT{Txt: []string{""}},
			}
			msg.Answer = append(msg.Answer, rr)
		case dns.TypeCAA: // Certificate Authority Authorization record
			rr := &dns.CAA{}
			msg.Answer = append(msg.Answer, rr)
		case dns.TypeANY:
		case dns.TypePTR: // PoinTeR
		case dns.TypeNS:
		case dns.TypeSOA:
		default: // nodata response
			soa := &dns.SOA{ // Start of Authority
				Hdr: h,
				SOA: rdata.SOA{
					Ns:      "",
					Mbox:    dnsutil.Join("hostmaster", dns.Zone(context.TODO())),
					Serial:  uint32(time.Now().Unix()),
					Minttl:  600,
					Refresh: 3600, Retry: 3600, Expire: 3600,
				},
			}
			msg.Ns = append(msg.Ns, soa)
		}
	}
	io.Copy(w, r)
}

// TODO!
func setupDNSserverLocally(
	cease_ch <-chan struct{}, // read-only
) {
	var should_terminate atomic.Bool

	should_terminate.Store(false)
	go func() {
		<-cease_ch
		should_terminate.Store(true)
	}()
	// serv := dns.NewServer()
	for !should_terminate.Load() {

	}
}

func TestLocalDNSserver(t *testing.T) {

	domain_arr := []string{
		"abcd.com",
		"alipay.com",
		"bilibili.com",
		"mit.edu",
		"www.nature.com",
		"journals.aps.org",
	}
	var cease_ch = make(chan struct{})

	go setupDNSserverLocally(cease_ch)
	for _, domain := range domain_arr {
		// any domain requires quary has to attach a dot after its domain
		domain = dnsutil.Fqdn(domain)
		res := QueryDNSRecord(domain, "localhost", dns.TypeAAAA)
		assert.NotEqual(t, res, nil)
		for _, r := range res {
			t.Log(r)
		}
		res = QueryDNSRecord(domain, "localhost", dns.TypeA)
		assert.NotEqual(t, res, nil)
		for _, r := range res {
			t.Log(r)
		}
		res = QueryDNSRecord(domain, "localhost", dns.TypeANY)
		assert.NotEqual(t, res, nil)
		for _, r := range res {
			t.Log(r)
		}
		res = QueryDNSRecord(domain, "localhost", dns.TypeCAA)
		assert.NotEqual(t, res, nil)
		for _, r := range res {
			t.Log(r)
		}
	}
	cease_ch <- struct{}{}
	close(cease_ch)
}
