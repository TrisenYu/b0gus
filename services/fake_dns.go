package services

import (
	"net"
	"sync"

	b0gus_config "b0gus/configs"
	b0gus_databases "b0gus/databases"
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
	addtional section 	variable, list of records

	header looks as follows:
		packet ID				16 bits
		Query Response			1  bit
		opcode					4  bits
		Authoritative Answer	1  bit
		Trancated Message		1  bit
		Recursion Desired		1  bit
		Recursion Available		1  bit
		Z(Reserved)				3  bits
		Response Code			4  bits
		Question Count			16 bits
		Answer Count			16 bits
		Authority Count			16 bits
		Additional Count 		16 bits
	question:
		name  lable sequence
		type  2 byte
		class 2 byte
*/

// TODO: What if we send a wrong response to other computer?

// Answers dns queries with a random ip address.
// Responds to versionbind queries with an old and unpatched version.

type DNSserverConf struct {
	/* database handler for writing data */
	DB_fd *b0gus_databases.RuntimeDB
	/* fields below need concurrenct control to follow the configuration */
	AlterDNSListener  sync.Mutex
	serverListenerPtr *net.UDPConn // current listener on Addr:Port
	Addr              string       // b0gus DNS server addr
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
		}
	}
	io.Copy(w, msg)
}

// db *gorm.DB *redis.Client *mongo.Client
func DNSserver(
	link_gadget *serviceReadCtrl,
	ntp_conf_obj *b0gus_config.DNSconfig,
	db *b0gus_databases.RuntimeDB,
	args ...any,
) {
	ntp_server_conf := DNSserverConf{
		Addr:  ntp_conf_obj.ListenAddr,
		Port:  ntp_conf_obj.ListenPort,
		DB_fd: db,
	}
	b0gus_config.Logger.Infof("%v", ntp_server_conf.Addr)
	// ntp_server_conf.NTPclientHandler(terminator)
}
