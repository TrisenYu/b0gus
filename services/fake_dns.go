package services

import (
	"context"
	"math/rand/v2"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
	"codeberg.org/miekg/dns/rdata"

	"b0gus/configs"
	"b0gus/databases"
)

// Reference: https://github.com/EmilHernvall/dnsguide/

// TODO: What if we send a wrong response to other computer?
// Answers dns queries with a random ip address.
// Responds to version bind queries with an old and unpatched version.

type DNSservConf struct {
	/* database handler for writing data */
	DBFd databases.DBhandler
	/* fields below need concurrent control to follow the configuration */
	ConfOptions *configs.DNSconfig
}

func getDNSrr(queryType uint16, headerName string) dns.RR {
	// [TODO]: we could set multiple answers for every query
	switch queryType {
	case dns.TypeA:
		var payload []byte
		for range 4 {
			payload = append(payload, byte(rand.IntN(256)))
		}
		return &dns.A{
			Hdr: dns.Header{
				Name:  headerName,
				Class: dns.ClassINET,
				TTL:   600,
			},
			A: rdata.A{Addr: netip.AddrFrom4([4]byte(payload))},
		}
	case dns.TypeAAAA:
		var payload []byte
		for range 16 {
			payload = append(payload, byte(rand.IntN(256)))
		}
		return &dns.AAAA{
			Hdr: dns.Header{
				Name:  headerName,
				Class: dns.ClassINET,
				TTL:   600,
			},
			AAAA: rdata.AAAA{Addr: netip.AddrFrom16([16]byte(payload))},
		}
	case dns.TypeCNAME:
		return &dns.CNAME{
			Hdr: dns.Header{
				Name:  headerName,
				Class: dns.ClassINET,
				TTL:   600,
			},
			CNAME: rdata.CNAME{
				Target: "cname." + headerName,
			},
		}
	case dns.TypeTXT:
		return &dns.TXT{
			Hdr: dns.Header{
				Name:  headerName,
				Class: dns.ClassINET,
				TTL:   600,
			},
			TXT: rdata.TXT{Txt: []string{"_acme-challenge." + headerName}},
		}
	case dns.TypeMX:
		return &dns.MX{
			Hdr: dns.Header{
				Name:  headerName,
				Class: dns.ClassINET,
				TTL:   600,
			},
			MX: rdata.MX{Mx: "mail." + headerName},
		}
	case dns.TypeNS:
		return &dns.NS{
			Hdr: dns.Header{
				Name:  headerName,
				Class: dns.ClassINET,
				TTL:   600,
			},
			NS: rdata.NS{Ns: "ns1." + headerName},
		}
	case dns.TypePTR:
		return &dns.PTR{
			Hdr: dns.Header{
				Name:  headerName,
				Class: dns.ClassINET,
				TTL:   600,
			},
			PTR: rdata.PTR{Ptr: "ptr1." + headerName},
		}
	case dns.TypeSRV:
		return &dns.SRV{
			Hdr: dns.Header{
				Name:  headerName,
				Class: dns.ClassINET,
				TTL:   600,
			},
			SRV: rdata.SRV{
				Priority: uint16(rand.IntN(65536)),
				Weight:   uint16(rand.IntN(65536)),
				Port:     uint16(rand.IntN(65536)),
				Target:   "cname." + headerName,
			},
		}
	default:
	}
	return nil
}

// analyzeQuery will record the remote request
func (d *DNSservConf) analyzeQuery(req dns.RR, m *dns.Msg) {
	headerName := req.Header().Name
	payload := databases.DnsQuery{
		DomainName: headerName,
		Opcode:     dnsutil.OpcodeToString(m.Opcode),
		QueryType:  dnsutil.TypeToString(dns.RRToType(req)),
	}
	_ = d.DBFd.CreateOrUpdateItem(&payload, &payload)
	answer := getDNSrr(dns.RRToType(req), headerName)
	if answer != nil {
		m.Answer = append(m.Answer, answer)
	} else {
		m.Rcode = dns.RcodeRefused
	}
}

// ServeDNS is implemented for the interface defined in miekg/dns
func (d *DNSservConf) ServeDNS(
	ctx context.Context,
	respWriter dns.ResponseWriter, r *dns.Msg,
) {
	m := new(dns.Msg)
	dnsutil.SetReply(m, r)

	addr, port, err := net.SplitHostPort(respWriter.RemoteAddr().String())
	if err == nil {
		p, err := strconv.Atoi(port)
		if err == nil {
			_ = d.DBFd.CreateOrUpdateItemsInSeq([]databases.DBstruct{
				&databases.AddrInfo{Ip: addr},
				&databases.PortInfo{Port: int64(p)},
			}...)
		} else {
			_ = d.DBFd.CreateOrUpdateItemsInSeq(&databases.AddrInfo{Ip: addr})
		}
	}
	for _, q := range m.Question {
		// TODO: ? multiple question should not affect the global section.
		d.analyzeQuery(q, m)
	}
	m.AuthenticatedData = true
	m.Authoritative = true
	m.Response = true
	_, err = m.WriteTo(respWriter)
	if err != nil {
		configs.Logger().Debug(err.Error())
	}
}

// Run will start up fake DNS server with recording
// every query requests
// [TODO]: since we use the framework, it is a little hard to switch the services
func (d *DNSservConf) Run(
	ConfObj *atomic.Pointer[configs.LocalConfig],
	scc *configs.ServConcurrentCtrl,
	db databases.DBhandler,
	args ...any,
) {
	defer func() {
		configs.Logger().Info(configs.GetLocalizedMsg("services.DNSQuitInfo", nil))
	}()
	if len(args) == 0 {
		configs.Logger().Error(configs.GetLocalizedMsg(
			"services.DNSWrongParamNumErr",
			map[string]any{
				"Expect": "> 0",
				"Actual": 0,
			},
		))
		return
	} else if ConfObj == nil {
		configs.Logger().Error(configs.GetLocalizedMsg("services.DNSNullConfErr", nil))
		return
	}
	snapshot, ok := ConfObj.Load().SelectTerm(configs.DNSEnum).(configs.DNSconfig)
	if !ok {
		configs.Logger().Error(configs.GetLocalizedMsg("services.DNSConfLoadErr", nil))
		return
	}
	var sb strings.Builder
	sb.WriteString(":")
	sb.WriteString(strconv.Itoa(int(snapshot.ListenPort)))
	s := &dns.Server{Addr: sb.String(), Net: "udp"}
	d.DBFd = db
	_ = d.DBFd.CreateTable(
		// have to re-create for certain table because services are separated
		&databases.AddrInfo{}, &databases.PortInfo{}, &databases.DnsQuery{},
	)
	var mu sync.Mutex // this mutex required by race condition detection.
	go func() {
		<-scc.Ctx.Done()
		mu.Lock()
		s.Shutdown(scc.Ctx)
		mu.Unlock()
	}()
	s.Handler = d
	mu.Lock()
	err := s.ListenAndServe()
	mu.Unlock()
	if err != nil {
		configs.Logger().Error(err.Error())
	}
}
