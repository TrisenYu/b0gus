// Package crypto_aux
package crypto_aux

// SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License
/// Last modified at 2026/05/16 星期六 12:12:32

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
	"io"
	"math/big"
	"net"
	"os"
	"strings"
	"time"

	"b0gus/configs"

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
		configs.Logger().Error(openPemFailure)
		return nil, err
	}
	defer func() { _ = pemFd.Close() }()
	pemBytes, err := io.ReadAll(pemFd)
	if err != nil {
		readPemFailure := configs.GetLocalizedMsg(
			"crypto_aux.PemFileReadFailure", nil,
		)
		// "Failed to read PEM file for SSH host key!"
		configs.Logger().Error(readPemFailure)
		return nil, err
	}
	pemBlock, _ := pem.Decode(pemBytes)
	if pemBlock == nil {
		pemDecodeFailure := configs.GetLocalizedMsg(
			"crypto_aux.PemFileDecodeFailure", nil,
		)
		configs.Logger().Error(pemDecodeFailure)
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

func handleEcdsa(pemLen uint64) (*ecdsa.PrivateKey, error) {
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
			map[string]any{"PemLen": pemLen},
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
		fallthrough
	case "ecdsa":
		hostPem, err = handleEcdsa(pemLen)
	case "rsa":
		hostPem, err = handleRsa(pemLen)
	default:
		errInfo := configs.GetLocalizedMsg(
			"crypto_aux.PemFileUnsupportedTypeError", nil,
		)
		configs.Logger().Error(errInfo)
		return nil, errors.New(errInfo)
	}
	return hostPem, err
}

// createPriKey will Generate a new host key when
// a PEM file given by configuration is invalid/corrupted or not exists
func createPriKey(
	pemPath, pemType string,
	pemLen uint64,
) (ssh.Signer, error) {
	hostPem, err := createPriObj(pemType, pemLen)
	if err != nil {
		payload := configs.GetLocalizedMsg(
			"crypto_aux.SSHPrivateKeyGenFailure",
			map[string]any{"ErrInfo": err},
		)
		configs.Logger().Error(payload)
		return nil, errors.New(payload)
	}
	hostKey, err := ssh.NewSignerFromKey(hostPem)
	if err != nil {
		payload := configs.GetLocalizedMsg(
			"crypto_aux.SSHPrivateKeySetFailure",
			map[string]any{"ErrInfo": err},
		)
		configs.Logger().Error(payload)
		return nil, errors.New(payload)
	}
	pemFile, err := os.Create(pemPath)
	if err != nil {
		payload := configs.GetLocalizedMsg(
			"crypto_aux.CreatePemToGivenPath",
			map[string]any{"PemPath": pemPath},
		)
		configs.Logger().Error(payload)
		return nil, errors.New(payload)
	}
	defer func() { _ = pemFile.Close() }()
	hostPemBytes, err := x509.MarshalPKCS8PrivateKey(hostPem)
	if err != nil {
		payload := configs.GetLocalizedMsg(
			"crypto_aux.PemFileDecodeFailure", nil,
		)
		configs.Logger().Error(payload)
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
		configs.Logger().Error(payload)
		return nil, errors.New(payload)
	}
	return hostKey, err
}

// LoadOrCreateSSHpem will try to load pem from pemPath
// if the previous actions failed, then the function will warn and attempt to creat one.
// When:
//  1. setting pemPath with only name, then the pem will be created under the same directory of the
//     invoked source file.
//  2. setting pemPath with available filepath, then the pem will be created under the assigned one
//
// Otherwise, this function return nil as its calculation result.
func LoadOrCreateSSHpem(
	pemPath, pemType string,
	pemLen uint64,
) ssh.Signer {
	pemObj, err := loadSSHhostPem(pemPath)
	if err == nil {
		return pemObj
	}
	res, err := createPriKey(pemPath, pemType, pemLen)
	if err == nil {
		// successfully generate one
		return res
	}
	payload := configs.GetLocalizedMsg(
		"crypto_aux.PemFileCreateFailure",
		map[string]any{
			"PemPath": pemPath,
			"ErrInfo": err,
		},
	)
	configs.Logger().Error(payload)
	return nil
}

type CAInfo struct {
	Cert       *smx509.Certificate
	PrivateKey crypto.PrivateKey
	KeyType    string
}

type CertSignConfig struct {
	pkix.Name
	Domains   []string
	IPs       []string
	KeyType   string
	KeySize   uint64
	ValidDays int
}

func CreateRootCaPair(
	certSignConf *CertSignConfig,
	caCertPath, caKeyPath string,
) error {
	rootCaKey, err := createPriObj(certSignConf.KeyType, certSignConf.KeySize)
	if err != nil {
		return err
	}
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return err
	}
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      certSignConf.Name,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(0, 0, certSignConf.ValidDays),
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment |
			x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            2,
		MaxPathLenZero:        false,
		DNSNames:              certSignConf.Domains,
		// [TODO]: OCSP server for checking and releasing certificates' states
	}
	rootCaCert, err := smx509.CreateCertificate(
		rand.Reader, template, template,
		rootCaKey.(crypto.Signer).Public(), rootCaKey,
	)
	if err != nil {
		return err
	}
	KeyBytes, err := smx509.MarshalPKCS8PrivateKey(rootCaKey)
	if err != nil {
		return err
	}
	certFd, err := os.Create(caCertPath)
	if err != nil {
		return err
	}
	defer func() { _ = certFd.Close() }()
	keyFd, err := os.Create(caKeyPath)
	if err != nil {
		return err
	}
	defer func() { _ = keyFd.Close() }()
	err = pem.Encode(keyFd, &pem.Block{Type: "PRIVATE KEY", Bytes: KeyBytes})
	if err != nil {
		return err
	}
	err = pem.Encode(certFd, &pem.Block{Type: "CERTIFICATE", Bytes: rootCaCert})
	return err
}

// LoadCA will load CAInfo from given caCertPath and caKeyPath.
func LoadCA(caCertPath, caKeyPath string) (*CAInfo, error) {
	caCertData, err := os.ReadFile(caCertPath)
	if err != nil {
		// TODO
		return nil, err
	}
	caCertBlock, _ := pem.Decode(caCertData)
	caCert, err := smx509.ParseCertificate(caCertBlock.Bytes)
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
		return nil, errors.New("invalid CA private key")
	}
	var (
		privateKey crypto.PrivateKey
		keyType    string
	)
	switch caKeyBlock.Type {
	case "RSA PRIVATE KEY":
		privateKey, err = x509.ParsePKCS1PrivateKey(caKeyBlock.Bytes)
		keyType = "rsa"
	case "PRIVATE KEY", "EC PRIVATE KEY":
		privateKey, err = smx509.ParsePKCS8PrivateKey(caKeyBlock.Bytes)
		switch privateKey.(type) {
		case *rsa.PrivateKey:
			keyType = "rsa"
		case *ecdsa.PrivateKey:
			keyType = "ecdsa"
		case *ed25519.PrivateKey:
			keyType = "ed25519"
		default:
			keyType = ""
		}
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

// IntranetSignCert is expected to sign local and trusted certificates.
// Nevertheless, this is a function violated against the open-closed principle
// due to the inconvenience of abstract/generic programming.
func IntranetSignCert(
	caInfo *CAInfo,
	refPem crypto.PrivateKey,
	SignedConf *CertSignConfig,
) (certPEM []byte, keyPEM []byte, err error) {
	// TODO: make errors info i18n
	if caInfo == nil || len(caInfo.KeyType) == 0 {
		return nil, nil, errors.New("CAInfo is nil")
	}
	if refPem == nil {
		refPem, err = createPriObj(SignedConf.KeyType, SignedConf.KeySize)
		if err != nil {
			return nil, nil, err
		}
	}

	// TODO: maintain the serialNumber
	// it will be much easier to maintain a increasing sequence of serialNumber
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, err
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      SignedConf.Name,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(0, 0, SignedConf.ValidDays),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
		IsCA:                  false,
		DNSNames:              SignedConf.Domains,
		// OCSPServer:
		// TODO: OCSP server for checking certificates' states
	}
	if len(SignedConf.IPs) > 0 {
		template.IPAddresses = make([]net.IP, len(SignedConf.IPs))
		for i, ip := range SignedConf.IPs {
			tmp := net.ParseIP(ip)
			if tmp == nil {
				continue
			}
			template.IPAddresses[i] = tmp
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
	switch SignedConf.KeyType {
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

// CreateCertPairUnderFilePath will write certData and KeyData
// into certPath and keyPath, respectively the certPath and keyPath should be
func CreateCertPairUnderFilePath(
	certPath, keyPath string,
	certData, keyData []byte,
	caData []byte,
) error {
	// TODO: i18n for error messages.
	certFd, err := os.Create(certPath)
	if err != nil {
		var errSb strings.Builder
		errSb.WriteString("can't not create certificate file due to: ")
		errSb.WriteString(err.Error())
		return errors.New(errSb.String())
	}
	defer func() { _ = certFd.Close() }()
	keyFd, err := os.Create(keyPath)
	if err != nil {
		var errSb strings.Builder
		errSb.WriteString("can't not create key file due to: ")
		errSb.WriteString(err.Error())
		return errors.New(errSb.String())
	}
	defer func() { _ = keyFd.Close() }()
	// just write bytes into fd and not care success or otherwise.
	_, _ = certFd.Write(append(certData, caData...))
	_, _ = keyFd.Write(keyData)
	return nil
}

// LoadLocalCertAsTLSServ will load signed (cert, key)-file for local tls-server and
// set rootCA as their certificate chain.
//
//	return nil if any error emerges.
func LoadLocalCertAsTLSServ(
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
	ok := RootCaCertPool.AppendCertsFromPEM(rootCaCertBytes)
	if !ok {
		// TODO: log it
		return nil
	}
	return &tls.Config{
		Certificates:           []tls.Certificate{cert},
		RootCAs:                RootCaCertPool,
		ClientCAs:              RootCaCertPool,
		ClientAuth:             tls.RequireAndVerifyClientCert,
		InsecureSkipVerify:     false,
		SessionTicketsDisabled: true,
		GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return &cert, nil
		},
	}
}

// LoadNormalCertAsTLSServ will load a signed (cert, key)-file for server,
// which is acknowledged by real-world CA.
func LoadNormalCertAsTLSServ(
	SignedCertFile, SignedKeyFile string,
) *tls.Config {
	cert, err := tls.LoadX509KeyPair(SignedCertFile, SignedKeyFile)
	if err != nil {
		// TODO may need logging
		return nil
	}
	return &tls.Config{
		Certificates:           []tls.Certificate{cert},
		ClientAuth:             tls.RequireAndVerifyClientCert,
		InsecureSkipVerify:     false,
		SessionTicketsDisabled: true,
		GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return &cert, nil
		},
	}
}

// LoadNormalCertAsTLSClient will load signed (cert, key)-file for client,
// which is acknowledged by real-world CA.
func LoadNormalCertAsTLSClient(
	SignedCertFile, SignedKeyFile, serverName string,
) *tls.Config {
	cert, err := tls.LoadX509KeyPair(SignedCertFile, SignedKeyFile)
	if err != nil {
		// TODO may need logging
		return nil
	}
	return &tls.Config{
		Certificates:           []tls.Certificate{cert},
		ServerName:             serverName,
		InsecureSkipVerify:     false,
		SessionTicketsDisabled: true,
	}
}

// LoadLocalCertAsTLSClient will load **trust** (cert, key) from files and return the most
// basic *tls.Config.
// In order to make local-sign (cert, key) trustable,
// RootCaCertPath is required for building up Chain of Certificate.
// For normal cert, use tls.LoadX509KeyPair is adequate.
func LoadLocalCertAsTLSClient(
	RootCaCertPath, SignedCertFile, SignedKeyFile string,
	serverName string,
) *tls.Config {
	cert, err := tls.LoadX509KeyPair(SignedCertFile, SignedKeyFile)
	if err != nil {
		// TODO need logging
		return nil
	}
	RootCaCertPool := x509.NewCertPool()
	rootCaCertBytes, err := os.ReadFile(RootCaCertPath)
	if err != nil {
		return nil
	}
	ok := RootCaCertPool.AppendCertsFromPEM(rootCaCertBytes)
	if !ok {
		return nil
	}
	return &tls.Config{
		Certificates:           []tls.Certificate{cert},
		RootCAs:                RootCaCertPool,
		ClientCAs:              RootCaCertPool,
		ServerName:             serverName,
		InsecureSkipVerify:     false,
		SessionTicketsDisabled: true,
	}
}
