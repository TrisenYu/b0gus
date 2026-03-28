package services

import (
	"b0gus/configs"
	"b0gus/databases"
	"context"
	"math/rand/v2"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"sync"

	"codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
	"codeberg.org/miekg/dns/rdata"
)

// Reference: https://github.com/EmilHernvall/dnsguide/

// TODO: What if we send a wrong response to other computer?

// Answers dns queries with a random ip address.
// Responds to version bind queries with an old and unpatched version.

type DNSserverConf struct {
	/* database handler for writing data */
	DBFd *configs.RuntimeDB
	/* fields below need concurrent control to follow the configuration */
	AlterDNSListener  sync.Mutex
	serverListenerPtr *net.UDPConn // current listener on Addr:Port
	ConfOptions       *configs.DNSconfig
}

func getDNSrr(queryType uint16, headerName string) dns.RR {
	// TODO: we could set multiple answers for every query
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
				Target: "cname.123.com",
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
			MX: rdata.MX{Mx: "mail."+headerName},
		}
	case dns.TypeNS:
		return &dns.NS{
			Hdr: dns.Header{
				Name:  headerName,
				Class: dns.ClassINET,
				TTL:   600,
			},
			NS: rdata.NS{Ns: "ns1."+headerName},
		}
	case dns.TypePTR:
		return &dns.PTR{
			Hdr: dns.Header{
				Name:  headerName,
				Class: dns.ClassINET,
				TTL:   600,
			},
			PTR: rdata.PTR{Ptr: "ptr1."+headerName},
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
				Target:   "cname."+headerName,
			},
		}
	default:
	}
	return nil
}

// analyze Query will record
func (d *DNSserverConf) analyzeQuery(req dns.RR, m *dns.Msg) {
	headerName := req.Header().Name
	payload := databases.DnsQuery{
		DomainName:         headerName,
		Opcode:             dnsutil.OpcodeToString(m.Opcode),
		QueryType:          dnsutil.TypeToString(dns.RRToType(req)),
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
func (d *DNSserverConf) ServeDNS(
	ctx context.Context,
	respWriter dns.ResponseWriter, r *dns.Msg,
) {
	m := new(dns.Msg)
	dnsutil.SetReply(m, r)

	addr, port, err := net.SplitHostPort(respWriter.RemoteAddr().String())
	if err == nil {
		p, err := strconv.Atoi(port)
		if err == nil {
			_ = d.DBFd.CreateOrUpdateItemsInSeq([]configs.DBstruct{
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
	m.Extra = append(m.Extra)
	m.AuthenticatedData = true
	m.Authoritative = true
	m.Response = true
	//m.Extra
	_, err = m.WriteTo(respWriter)
	if err != nil {
		configs.Logger.Debug(err.Error())
	}
}

// Run will start up fake DNS server with recording
// every query requests
func (d *DNSserverConf) Run(
	dnsConfObj *configs.DNSconfig,
	scc *configs.ServConcurrentCtrl,
	db *configs.RuntimeDB, // db *gorm.DB *redis.Client *mongo.Client
	args ...any,
) {
	defer func() { configs.Logger.Info("DNS server quit...") }()
	if len(args) == 0 {
		configs.Logger.Error("incorrect number of arguments")
		return
	} else if dnsConfObj == nil {
		configs.Logger.Error("empty configuration is provided")
		return
	}
	var sb strings.Builder
	sb.WriteString(":")
	sb.WriteString(strconv.Itoa(int(dnsConfObj.ListenPort)))
	s := &dns.Server{Addr: sb.String(), Net: "udp"}
	d.DBFd = db
	_ = d.DBFd.CreateTable(
		&databases.AddrInfo{},
		&databases.PortInfo{}, // have to re-create because services is separated
		&databases.DnsQuery{},
	)
	go func() {
		<-scc.Ctx.Done()
		s.Shutdown(nil)
	}()
	s.Handler = d
	err := s.ListenAndServe()
	if err != nil {
		configs.Logger.Error(err.Error())
	}
}
