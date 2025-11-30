package services

import (
	"net"
	"sync"

	b0gus_config "b0gus/configs"
	b0gus_databases "b0gus/databases"
	b0gus_datatypes "b0gus/generic_datatypes"
)

// Reference: https://github.com/EmilHernvall/dnsguide/

// Answers dns queries with a random ip address.
// Responds to versionbind queries with an old and unpatched version.

type DNSserverConf struct {
	/* database handler for writing data */
	DB_fd *b0gus_databases.RecordDB
	/* fields below need concurrenct control to follow the configuration */
	AlterDNSListener  sync.Mutex
	ConfigGenericCtrl b0gus_datatypes.ConcurrentCtrl
	serverListenerPtr *net.UDPConn // current listener on Addr:Port
	Addr              string       // b0gus NTP server addr
	Port              uint16       // b0gus NTP server port number
}

// db *gorm.DB *redis.Client *mongo.Client
func DNSserver(
	terminator *b0gus_datatypes.ConcurrentCtrl,
	ntp_conf_obj *b0gus_config.DNSconfig,
	db *b0gus_databases.RecordDB,
	wait_group *sync.WaitGroup,
) {
	defer wait_group.Done()
	ntp_server_conf := DNSserverConf{
		Addr:  ntp_conf_obj.ListenAddr,
		Port:  ntp_conf_obj.ListenPort,
		DB_fd: db,
	}
	ntp_server_conf.ConfigGenericCtrl.Ch = make(chan struct{}, 1)
	ntp_server_conf.ConfigGenericCtrl.Flag.Store(false)

	// ntp_server_conf.NTPclientHandler(terminator)
	close(ntp_server_conf.ConfigGenericCtrl.Ch)

}
