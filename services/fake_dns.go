package services

import (
	"fmt"
	"net"
	"sync"

	"b0gus/configs"
	"io"

	"codeberg.org/miekg/dns"
)

// Reference: https://github.com/EmilHernvall/dnsguide/
// origin DNS packet arranges its structure in such a way:
/*
	header 				12 bytes
	------------------------------------------------
	question section	variable, list of questions
	------------------------------------------------
	answer section 		variable, list of records
	authority section	variable, list of records
	additional section 	variable, list of records

	header looks as follows:
		packet ID				16 bits
		Query Response			1  bit
		opcode					4  bits
		Authoritative Answer	1  bit
		Truncated Message		1  bit
		Recursion Desired		1  bit
		Recursion Available		1  bit
		Z(Reserved)				3  bits
		Response Code			4  bits
		Question Count			16 bits
		Answer Count			16 bits
		Authority Count			16 bits
		Additional Count 		16 bits
	question:
		name  label sequence
		type  2 byte
		class 2 byte
*/

// TODO: What if we send a wrong response to other computer?

// Answers dns queries with a random ip address.
// Responds to version bind queries with an old and unpatched version.

type DNSserverConf struct {
	/* database handler for writing data */
	DBFd *configs.RuntimeDB
	/* fields below need concurrent control to follow the configuration */
	AlterDNSListener  sync.Mutex
	serverListenerPtr *net.UDPConn // current listener on Addr:Port
	Port              uint16       // b0gus DNS server port number
}

func handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	msg := r.Copy()
	msg.Authoritative = true

	for _, question := range r.Question {
		switch question.Header().Class {
		case dns.TypeA:
			// handleARecord(question, msg)
		case dns.TypeAAAA:
			// handleAAAARecord(question, msg)
		case dns.TypeTXT:
			// handleTextRecord()
		case dns.TypeCNAME:
		case dns.TypeNS:
		default:

		}
	}
	_, _ = io.Copy(w, msg)
}

func Run(
	ntpConfObj *configs.DNSconfig,
	scc *configs.ServConcurrentCtrl,
	db *configs.RuntimeDB, // db *gorm.DB *redis.Client *mongo.Client
	args ...any,
) {
	ntpServerConf := DNSserverConf{
		Port: ntpConfObj.ListenPort,
		DBFd: db,
	}
	payload := fmt.Sprintf("%v", ntpServerConf.Port)
	configs.Logger.Info(payload)
	// DNSclientHandler(terminator)
}
