package diff_tests

import (
	"b0gus/configs"
	"b0gus/crypto_aux"
	"context"
	"crypto/tls"
	"crypto/x509/pkix"
	"encoding/pem"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ssh"
)

func TestPemHelper(t *testing.T) {
	var (
		pemPath = "./test_dir/test.pem"
		delPem  = func() {
			err := os.Remove(pemPath)
			assert.Equal(t, nil, err, "Unable to delete test.pem")
		}
	)

	whatWeHave := crypto_aux.LoadOrCreateSSHpem(pemPath, "ed25519", 0)
	assert.NotEqual(t, nil, whatWeHave, "Still got an nil after creating")
	delPem()
	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "ecdsa", 311)
	assert.Equal(t, nil, whatWeHave, "unexpected nil pem")

	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "ecdsa", 521)
	assert.NotEqual(t, nil, whatWeHave, "unexpected nil pem")
	delPem()

	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "ecdsa", 384)
	assert.NotEqual(t, nil, whatWeHave, "unexpected nil pem")
	delPem()

	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "ecdsa", 0)
	assert.Equal(t, nil, whatWeHave, "unexpected nil pem")

	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "ell1ptic", 0)
	assert.Equal(t, nil, whatWeHave, "unexpected nil pem")

	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "rsa", 384)
	assert.Equal(t, nil, whatWeHave, "unexpected nil pem")

	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "rsa", 1024)
	assert.NotEqual(t, nil, whatWeHave, "unexpected nil pem")
	delPem()

	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "rsa", 2048)
	assert.NotEqual(t, nil, whatWeHave, "unexpected nil pem")
	delPem()

	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "rsa", 4096)
	assert.NotEqual(t, nil, whatWeHave, "unexpected nil pem")

	tmp := whatWeHave.(any)
	_, ok := tmp.(ssh.Signer)
	assert.Equal(t, true, ok)

	delPem()
	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "rsa", 1234)
	assert.Equal(t, nil, whatWeHave, "unexpected nil pem")

	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "ed25519", 123456)
	assert.NotEqual(t, nil, whatWeHave, "unexpected nil pem")
	whatWeHave = nil
	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "ed25519", 123456)
	assert.NotEqual(t, nil, whatWeHave, "unexpected nil pem")
	delPem()

	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "ecdh", 123456)
	assert.Equal(t, nil, whatWeHave, "Still got an nil after creating")

	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "dsa", 123456)
	assert.Equal(t, nil, whatWeHave, "Still got an nil after creating")

	whatWeHave = crypto_aux.LoadOrCreateSSHpem(pemPath, "sm2", 123456)
	assert.Equal(t, nil, whatWeHave, "Still got an nil after creating")
}

func TestLocalCertSigning(t *testing.T) {
	certKeyDir := "./test_dir/"
	certPath := certKeyDir + "tmpCa.cert"
	keyPath := certKeyDir + "tmpCa.key"
	err := crypto_aux.CreateRootCaPair(
		&crypto_aux.CertSignConfig{
			Name:      pkix.Name{CommonName: "a.bcd"},
			Domains:   []string{"a.bcd"},
			IPs:       []string{"127.0.0.1", "127.0.0.2"},
			ValidDays: 2,
			KeyType:   "ecdsa",
			KeySize:   521,
		},
		certPath, keyPath,
	)
	assert.NoError(t, err)
	caInfo, err := crypto_aux.LoadCA(certPath, keyPath)
	assert.NoError(t, err)
	assert.NotNil(t, caInfo)
	cert1, key1, err := crypto_aux.IntranetSignCert(
		caInfo, nil,
		&crypto_aux.CertSignConfig{
			Name:      pkix.Name{CommonName: "*.xyz.a.bcd"},
			IPs:       []string{"127.0.0.1", "127.0.0.2"},
			Domains:   []string{"www.xyz.a.bcd"},
			ValidDays: 2,
			KeyType:   "ed25519",
			KeySize:   256,
		},
	)
	assert.NoError(t, err)
	cert2, key2, err := crypto_aux.IntranetSignCert(
		caInfo, nil,
		&crypto_aux.CertSignConfig{
			Name:      pkix.Name{CommonName: "*.xyz.a.bcd"},
			IPs:       []string{"127.0.0.1", "127.0.0.2"},
			Domains:   []string{"git.xyz.a.bcd"},
			ValidDays: 2,
			KeyType:   "ed25519",
			KeySize:   384,
		},
	)
	assert.NoError(t, err)
	err = crypto_aux.CreateCertPairUnderFilePath(
		certKeyDir+"client1.cert", certKeyDir+"client1.key",
		cert1, key1, pem.EncodeToMemory(
			&pem.Block{Type: "CERTIFICATE", Bytes: caInfo.Cert.Raw},
		),
	)
	assert.NoError(t, err)
	err = crypto_aux.CreateCertPairUnderFilePath(
		certKeyDir+"client2.cert", certKeyDir+"client2.key",
		cert2, key2, pem.EncodeToMemory(
			&pem.Block{Type: "CERTIFICATE", Bytes: caInfo.Cert.Raw},
		),
	)
	assert.NoError(t, err)

	// stage 3: set up tls connections
	var (
		wg          sync.WaitGroup
		ctx, cancel = context.WithCancel(context.Background())
	)
	// before running test, we have to register the root CA into the system
	configs.RemotePullSource = ":45678"
	wg.Go(func() {
		// client1 as server
		crypto_aux.PullUpdatesFromRemote(
			certKeyDir+"tmpCa.cert",
			certKeyDir+"client1.cert",
			certKeyDir+"client1.key",
			ctx,
		)
	})
	wg.Go(func() {
		// client2 as client
		defer cancel()
		time.Sleep(2 * time.Second)
		conn, err := tls.Dial(
			"tcp", configs.RemotePullSource,
			crypto_aux.LoadLocalCertAsTLSClient(
				certKeyDir+"tmpCa.cert",
				certKeyDir+"client2.cert",
				certKeyDir+"client2.key",
				"www.xyz.a.bcd", // well, preknown domain.
			),
		)
		if err != nil {
			assert.FailNow(t, err.Error())
			return
		}
		_, _ = conn.Write([]byte("?!?hello world!?!"))
	})
	wg.Wait()
	ctx, cancel = context.WithCancel(context.Background())
	wg.Go(func() {
		// client1 as server
		crypto_aux.PullUpdatesFromRemote(
			certKeyDir+"tmpCa.cert",
			certKeyDir+"client1.cert",
			certKeyDir+"client1.key",
			ctx,
		)
	})
	wg.Go(func() {
		// client2 as client
		defer cancel()
		time.Sleep(2 * time.Second)
		conn, err := tls.Dial(
			"tcp", configs.RemotePullSource,
			crypto_aux.LoadLocalCertAsTLSClient(
				certKeyDir+"tmpCa.cert",
				certKeyDir+"client2.cert",
				certKeyDir+"client2.key",
				"127.0.0.1", // we can use IP once we properly signature the certificate
			),
		)
		if err != nil {
			assert.FailNow(t, err.Error())
			return
		}
		_, _ = conn.Write([]byte("whoa!"))
	})
	wg.Wait()
	// TODO: withdraw the registered root CA
}
