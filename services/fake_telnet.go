// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
package services

import (
	"sync"

	b0gus_config "b0gus/configs"
	b0gus_databases "b0gus/databases"
)

// reference online documentation: https://www.rfc-editor.org/rfc/rfc854
// default port of telnet is 23
type Telnet struct {
}

// db *gorm.DB *redis.Client *mongo.Client
func TelnetServer(
	link_gadget *serviceReadCtrl,
	bogus_conf *b0gus_config.TelnetConfig,
	db *b0gus_databases.RecordDB,
	wait_group *sync.WaitGroup,
	args ...any,
) {
	wait_group.Done()
}
