package diff_tests

import (
	"b0gus/crypto_aux"
	"crypto/x509/pkix"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ssh"
)

func TestPemHelper(t *testing.T) {

	var (
		pemPath = "../assets/test.pem"
		delPem  = func() {
			err := os.Remove(pemPath)
			assert.Equal(t, err, nil, "Unable to delete test.pem")
		}
	)

	whatWeHave := crypto_aux.LoadOrCreateSSHpem(pemPath, "ed25519", 0)
	assert.NotEqual(t, whatWeHave, nil, "Still got an nil after creating")
	delPem()
	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "elliptic", 311)
	assert.Equal(t, whatWeHave, nil, "unexpected nil pem")

	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "elliptic", 521)
	assert.NotEqual(t, whatWeHave, nil, "unexpected nil pem")
	delPem()

	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "elliptic", 384)
	assert.NotEqual(t, whatWeHave, nil, "unexpected nil pem")
	delPem()

	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "elliptic", 0)
	assert.Equal(t, whatWeHave, nil, "unexpected nil pem")

	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "ell1ptic", 0)
	assert.Equal(t, whatWeHave, nil, "unexpected nil pem")

	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "rsa", 384)
	assert.Equal(t, whatWeHave, nil, "unexpected nil pem")

	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "rsa", 1024)
	assert.NotEqual(t, whatWeHave, nil, "unexpected nil pem")
	delPem()

	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "rsa", 2048)
	assert.NotEqual(t, whatWeHave, nil, "unexpected nil pem")
	delPem()

	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "rsa", 4096)
	assert.NotEqual(t, whatWeHave, nil, "unexpected nil pem")

	tmp := whatWeHave.(any)
	_, ok := tmp.(ssh.Signer)
	assert.Equal(t, true, ok)

	delPem()
	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "rsa", 1234)
	assert.Equal(t, whatWeHave, nil, "unexpected nil pem")

	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "ed25519", 123456)
	assert.NotEqual(t, whatWeHave, nil, "unexpected nil pem")
	whatWeHave = nil
	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "ed25519", 123456)
	assert.NotEqual(t, whatWeHave, nil, "unexpected nil pem")
	delPem()

	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "ecdh", 123456)
	assert.Equal(t, whatWeHave, nil, "Still got an nil after creating")

	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "dsa", 123456)
	assert.Equal(t, whatWeHave, nil, "Still got an nil after creating")

	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "sm2", 123456)
	assert.Equal(t, whatWeHave, nil, "Still got an nil after creating")
}

func TestLocalCertSigning(t *testing.T) {
	certKeyDir := "./test_dir/"
	caInfo, err := crypto_aux.LoadCA(certKeyDir+"abc.cert", certKeyDir+"abc.key")
	assert.NoError(t, err)
	assert.NotNil(t, caInfo)
	certPem, keyPem, err := crypto_aux.IntranetSignCert(
		caInfo, nil,
		&crypto_aux.CertSignConfig{
			Name:      pkix.Name{CommonName: "helo.world"},
			IPs:       []string{"192.168.1.0", "192.168.1.1"},
			Domains:   []string{"xyz.abc.d"},
			ValidDays: 30,
			KeySize:   512,
		},
	)
	assert.NoError(t, err)
	fmt.Println(string(certPem), string(keyPem))
}
