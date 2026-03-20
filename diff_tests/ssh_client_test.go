// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
package diff_tests

import (
	"testing"
	// assert "github.com/stretchr/testify/assert"
)

type MockSSHConn struct {
	Username, Password string
	PublicKey          string
	Commands           []string
}

func TestGenDefForSSH(t *testing.T) {
	// ?
}

func TestSSHclient(t *testing.T) {
	// TODO
}
