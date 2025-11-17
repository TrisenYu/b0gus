package crypto_aux

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"os"

	b0gus_config "b0gus/configs"

	"golang.org/x/crypto/ssh"
)

/*- Load ssh pem from given path
 * inParam: pem_path string; the path of the pem file, invisible to remote address
 * return ssh.Signer, error; if success, return ssh.Signer object and nil, else return nil and error
 */
func LoadHostPem(pem_path string) (ssh.Signer, error) {
	pem_fd, err := os.Open(pem_path)
	if err != nil {
		b0gus_config.Logger.WithField("pem_path", pem_path).Error("Failed to open PEM file for SSH host key!\n")
		return nil, err
	}
	pem_bytes, err := io.ReadAll(pem_fd)
	if err != nil {
		b0gus_config.Logger.Error("Failed to read PEM file for SSH host key!\n")
		return nil, err
	}
	pem_fd.Close()
	pem_block, _ := pem.Decode(pem_bytes)
	if pem_block == nil {
		b0gus_config.Logger.Error("Failed to decode PEM block for SSH host key!\n")
		return nil, err
	}

	switch pem_block.Type {
	case "RSA PRIVATE KEY":
		rsa_private, err := x509.ParsePKCS1PrivateKey(pem_block.Bytes)
		if rsa_private == nil || err != nil {
			return nil, err
		}
		return ssh.NewSignerFromKey(rsa_private)
	case "EC PRIVATE KEY":
		// nil pointer error
		ecc_private, err := x509.ParseECPrivateKey(pem_block.Bytes)
		if ecc_private == nil || err != nil {
			return nil, err
		}
		return ssh.NewSignerFromKey(ecc_private)
	case "PRIVATE KEY":
		any_private, err := x509.ParsePKCS8PrivateKey(pem_block.Bytes)
		if any_private == nil || err != nil {
			return nil, err
		}
		return ssh.NewSignerFromKey(any_private)
	default:
		return nil, fmt.Errorf("unsupported signing pem format")
	}
}
