// SPDX-LICENSE-IDENTIFIER: 3-Clause-BSD
package services

import (
	"sync"

	gorm "gorm.io/gorm"

	b0gus_config "b0gus/configs"
	b0gus_datatypes "b0gus/datatypes"
)

// reference online documentation: https://www.rfc-editor.org/rfc/rfc854
// default port of telnet is 23, and end of line is \r\n
type Telnet struct {
}

func TelnetServer(
	need_shutdown *b0gus_datatypes.ConcurrentCtrl,
	bogus_conf *b0gus_config.TelnetConfig,
	db *gorm.DB,
	wait_group *sync.WaitGroup,
) {
	wait_group.Done()
}
