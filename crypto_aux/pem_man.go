// Package crypto_aux
package crypto_aux

// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"strings"
	"time"

	"b0gus/configs"

	"github.com/emmansun/gmsm/sm2"
	"github.com/emmansun/gmsm/smx509"
	"golang.org/x/crypto/ssh"
)

// loadSSHhostPem will Load ssh pem from given path
// inParam: pemPath string; the path of the pem file, invisible to remote address
// return ssh.Signer, error; if success, return ssh.Signer object and nil,
// else return nil and error
func loadSSHhostPem(pemPath string) (ssh.Signer, error) {
	pemFd, err := os.Open(pemPath)
	if err != nil {
		openPemFailure := configs.GetLocalizedMsg(
			"crypto_aux.PemFileOpenFailure",
			map[string]any{"PemPath": pemPath},
		)
		// "Failed to read PEM file for SSH host key!"
		configs.Logger.Error(openPemFailure)
		return nil, err
	}
	defer func() { _ = pemFd.Close() }()
	pemBytes, err := io.ReadAll(pemFd)
	if err != nil {
		readPemFailure := configs.GetLocalizedMsg(
			"crypto_aux.PemFileReadFailure", nil,
		)
		// "Failed to read PEM file for SSH host key!"
		configs.Logger.Error(readPemFailure)
		return nil, err
	}
	pemBlock, _ := pem.Decode(pemBytes)
	if pemBlock == nil {
		pemDecodeFailure := configs.GetLocalizedMsg(
			"crypto_aux.PemFileDecodeFailure", nil,
		)
		configs.Logger.Error(pemDecodeFailure)
		return nil, err
	}
	switch pemBlock.Type {
	case "RSA PRIVATE KEY":
		rsaPrivate, err := x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
		if rsaPrivate == nil || err != nil {
			return nil, err
		}
		return ssh.NewSignerFromKey(rsaPrivate)
	case "EC PRIVATE KEY":
		// nil pointer error
		eccPrivate, err := x509.ParseECPrivateKey(pemBlock.Bytes)
		if eccPrivate == nil || err != nil {
			return nil, err
		}
		return ssh.NewSignerFromKey(eccPrivate)
	case "PRIVATE KEY":
		anyPrivate, err := x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
		if anyPrivate == nil || err != nil {
			return nil, err
		}
		return ssh.NewSignerFromKey(anyPrivate)
	default:
		pemFileFormatErr := configs.GetLocalizedMsg(
			"crypto_aux.PemFileFormatError", nil,
		)
		return nil, errors.New(pemFileFormatErr)
	}
}

// handleEd25519 will get its result from `ed25519.GenerateKey(rand.Reader)`,
// while aborting the public key since we can calculate the public key via
// private key.
func handleEd25519() (*ed25519.PrivateKey, error) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	return &privateKey, err
}

func handleElliptic(pemLen uint64) (*ecdsa.PrivateKey, error) {
	var choice elliptic.Curve
	switch pemLen {
	case 224:
		choice = elliptic.P224()
	case 256:
		choice = elliptic.P256()
	case 384:
		choice = elliptic.P384()
	case 521:
		choice = elliptic.P521()
	default:
		invalidLen := configs.GetLocalizedMsg(
			"crypto_aux.InvalidLengthForEllipticCurve",
			map[string]any{
				"PemLen": pemLen,
			},
		)
		return nil, errors.New(invalidLen)
	}
	return ecdsa.GenerateKey(choice, rand.Reader)
}

func handleRsa(pemLen uint64) (*rsa.PrivateKey, error) {
	switch pemLen {
	case 1024:
		// this is not secure at all. Use 2048 instead
		pemLen = 2048
	case 2048:
	case 4096:
	case 8192:
	default:
		invalidLen := configs.GetLocalizedMsg(
			"crypto_aux.InvalidLengthForRSA",
			map[string]any{
				"PemLen": pemLen,
			},
		)
		return nil, errors.New(invalidLen)
	}
	return rsa.GenerateKey(rand.Reader, int(pemLen&0xFFFF_FFFF))
}

// createPriObj will generate a private key
func createPriObj(pemType string, pemLen uint64) (crypto.PrivateKey, error) {
	var (
		hostPem any = nil
		err     error
	)
	switch strings.ToLower(pemType) {
	case "ed25519":
		currPem, err := handleEd25519()
		if err == nil && currPem != nil {
			hostPem = *currPem
		}
	case "elliptic":
		hostPem, err = handleElliptic(pemLen)
	case "rsa":
		hostPem, err = handleRsa(pemLen)
	case "sm2":
		hostPem, err = sm2.GenerateKey(rand.Reader)
	default:
		errInfo := configs.GetLocalizedMsg(
			"crypto_aux.PemFileUnsupportedTypeError", nil,
		)
		configs.Logger.Error(errInfo)
		return nil, errors.New(errInfo) // fmt.Errorf("invalid pem type:<%v>", pem_type)
	}
	return hostPem, err
}

// createPriKey will Generate a new host key when
// a PEM file given by configuration is invalid/corrupted or not exists
func createPriKey(
	pemPath string,
	pemType string,
	pemLen uint64,
) (ssh.Signer, error) {
	hostPem, err := createPriObj(pemType, pemLen)
	if err != nil {
		payload := configs.GetLocalizedMsg(
			"crypto_aux.SSHPrivateKeyGenFailure",
			map[string]any{"ErrInfo": err},
		)
		configs.Logger.Error(payload)
		return nil, errors.New(payload)
	}
	hostKey, err := ssh.NewSignerFromKey(hostPem)
	if err != nil {
		payload := configs.GetLocalizedMsg(
			"crypto_aux.SSHPrivateKeySetFailure",
			map[string]any{"ErrInfo": err},
		)
		configs.Logger.Error(payload)
		return nil, errors.New(payload)
	}
	pemFile, err := os.Create(pemPath)
	if err != nil {
		payload := configs.GetLocalizedMsg(
			"crypto_aux.CreatePemToGivenPath",
			map[string]any{"PemPath": pemPath},
		)
		configs.Logger.Error(payload)
		return nil, errors.New(payload)
	}
	defer func() { _ = pemFile.Close() }()
	hostPemBytes, err := x509.MarshalPKCS8PrivateKey(hostPem)
	if err != nil {
		payload := configs.GetLocalizedMsg(
			"crypto_aux.PemFileDecodeFailure", nil,
		)
		configs.Logger.Error(payload)
		return nil, errors.New(payload)
	}
	hostPemBlock := pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: hostPemBytes,
	}
	err = pem.Encode(pemFile, &hostPemBlock)
	if err != nil {
		payload := configs.GetLocalizedMsg(
			"crypto_aux.Base64EncodeFailure", nil,
		)
		configs.Logger.Error(payload)
		return nil, errors.New(payload)
	}
	return hostKey, err
}

// LoadOrCreateSSHpem will try to load pem from pemPath
// if the previous actions failed, then the function will warn and attempt to creat one.
func LoadOrCreateSSHpem(
	pemPath string,
	pemType string,
	pemLen uint64,
) ssh.Signer {
	pemObj, err := loadSSHhostPem(pemPath)
	if err == nil {
		return pemObj
	}
	var res ssh.Signer
	payload := configs.GetLocalizedMsg(
		"crypto_aux.UnsupportedOrInvalidPemWarn",
		map[string]any{"ErrInfo": err},
	)
	configs.Logger.Warn(payload)
	res, err = createPriKey(pemPath, pemType, pemLen)
	if err == nil {
		// successfully generate one
		return res
	}
	payload = configs.GetLocalizedMsg(
		"crypto_aux.PemFileCreateFailure",
		map[string]any{
			"PemPath": pemPath,
			"ErrInfo": err,
		},
	)
	configs.Logger.Error(payload)
	return nil
}

type CAInfo struct {
	Cert       *x509.Certificate
	PrivateKey crypto.PrivateKey
	KeyType    string
}

// LoadCA will load CAInfo from given caCertPath and caKeyPath.
func LoadCA(caCertPath, caKeyPath string) (*CAInfo, error) {
	caCertData, err := os.ReadFile(caCertPath)
	if err != nil {
		// TODO
		return nil, err
	}
	caCertBlock, _ := pem.Decode(caCertData)
	caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		// TODO
		return nil, err
	}
	caKeyData, err := os.ReadFile(caKeyPath)
	if err != nil {
		// TODO
		return nil, err
	}
	caKeyBlock, _ := pem.Decode(caKeyData)
	if caKeyBlock == nil {
		return nil, fmt.Errorf("invalid CA private key")
	}
	var (
		privateKey crypto.PrivateKey
		keyType    string
	)
	switch caKeyBlock.Type {
	case "RSA PRIVATE KEY":
		privateKey, err = x509.ParsePKCS1PrivateKey(caKeyBlock.Bytes)
		keyType = "rsa"
	case "PRIVATE KEY":
		privateKey, err = x509.ParsePKCS8PrivateKey(caKeyBlock.Bytes)
		switch privateKey.(type) {
		case *rsa.PrivateKey:
			keyType = "rsa"
		case *ecdsa.PrivateKey:
			keyType = "ecdsa"
		case ed25519.PrivateKey:
			keyType = "ed25519"
		default:
			keyType = ""
		}
	case "EC PRIVATE KEY":
		privateKey, err = smx509.ParseSM2PrivateKey(caKeyBlock.Bytes)
		keyType = "sm2"
	default:
		privateKey = nil
		keyType = ""
	}

	return &CAInfo{
		Cert:       caCert,
		PrivateKey: privateKey,
		KeyType:    keyType,
	}, err
}

type CertSignConfig struct {
	pkix.Name
	Domains   []string
	IPs       []string
	ValidDays int    //
	KeySize   uint64 //
}

// IntranetSignCert is expected to sign local and trusted certificates.
// Nevertheless, this is a function violated against the open-closed principle
// due to the inconvenience of abstract/generic programming.
//
//	caInfo, err := LoadCA("/path/to/ca.crt", "/path/to/ca.key")
//	if err != nil {
//		log.Fatal(err)
//	}
//	signConfig := CertSignConfig{
//		CertCommonName:     "internal-service.example.com",
//		IPs:        []string{"192.168.1.123"},
//		Domains:    []string{"service.internal"},
//		ValidDays:  3650,
//		KeySize:    256,
//	}
//	certPEM, keyPEM, err := GenerateSignedCert(caInfo, signConfig)
//	_ = os.WriteFile("abc.crt", certPEM, 0644)
//	_ = os.WriteFile("abc.key", keyPEM, 0600)
func IntranetSignCert(
	caInfo *CAInfo,
	refPem crypto.PrivateKey,
	certSignConf *CertSignConfig,
) (certPEM []byte, keyPEM []byte, err error) {

	// TODO: make errors info i18n
	if caInfo == nil || len(caInfo.KeyType) == 0 {
		return nil, nil, errors.New("CAInfo is nil")
	}
	if refPem == nil {
		refPem, err = createPriObj(caInfo.KeyType, certSignConf.KeySize)
		if err != nil {
			return nil, nil, err
		}
	}

	// TODO: maintain the serialNumber
	// it will be much more easier to maintain a increasing sequence of serialNumber
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, err
	}

	template := &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               certSignConf.Name,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, certSignConf.ValidDays),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
		DNSNames:              certSignConf.Domains,
		// OCSPServer:
		// TODO: OCSP server for checking certificates' states
	}
	if len(certSignConf.IPs) > 0 {
		template.IPAddresses = make([]net.IP, len(certSignConf.IPs))
		for i, ip := range certSignConf.IPs {
			template.IPAddresses[i] = net.ParseIP(ip)
		}
	}
	// sign certificate here
	var certBytes []byte
	tmpPriKey, ok := refPem.(crypto.Signer)
	if !ok {
		return nil, nil, errors.New("invalid private key")
	}
	certBytes, err = smx509.CreateCertificate(
		rand.Reader, template, caInfo.Cert,
		tmpPriKey.Public(), caInfo.PrivateKey,
	)
	if err != nil {
		return nil, nil, err
	}
	switch caInfo.KeyType {
	case "sm2":
		pemAsBytes, _ := refPem.(*sm2.PrivateKey).Bytes()
		keyPEM = pem.EncodeToMemory(&pem.Block{
			Type:  "SM2 PRIVATE KEY",
			Bytes: pemAsBytes,
		})
	case "rsa":
		tmpCast, ok := refPem.(*rsa.PrivateKey)
		if !ok {
			return nil, nil, errors.New(
				"unable to cast file into rsa private key",
			)
		}
		keyPEM = pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(tmpCast),
		})
	case "ed25519":
		fallthrough
	case "ecdsa":
		pemAsBytes, _ := x509.MarshalPKCS8PrivateKey(refPem)
		keyPEM = pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: pemAsBytes,
		})
	default:
		return nil, nil, errors.New("encounter an unsupported key type")
	}
	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	// after returning result, the first thing has to do is to record them
	return certPEM, keyPEM, nil
}

// LoadLocalCert will load **trust** (cert, key) from files and return the most
// basic *tls.Config.
// In order to make local-sign (cert, key) as trust, RootCaCertPath is required.
// For normal cert, use tls.LoadX509KeyPair is adequate.
func LoadLocalCert(
	RootCaCertPath, SignedCertFile, SignedKeyFile string,
) *tls.Config {
	cert, err := tls.LoadX509KeyPair(SignedCertFile, SignedKeyFile)
	if err != nil {
		// TODO may need logging
		return nil
	}
	RootCaCertPool := x509.NewCertPool()
	rootCaCertBytes, err := os.ReadFile(RootCaCertPath)
	if err != nil {
		return nil
	}
	RootCaCertPool.AppendCertsFromPEM(rootCaCertBytes)
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    RootCaCertPool,
	}
}
