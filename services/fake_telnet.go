// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
package services

import (
	b0gus_config "b0gus/configs"
)

// reference online documentation: https://www.rfc-editor.org/rfc/rfc854
// default port of telnet is 23
type Telnet struct {
}

// db *gorm.DB *redis.Client *mongo.Client
func TelnetServer(
	bogus_conf *b0gus_config.TelnetConfig,
	scc *b0gus_config.ServicesConcurrencyCtrl,
	db *b0gus_config.RuntimeDB,
	args ...any,
) {
}
