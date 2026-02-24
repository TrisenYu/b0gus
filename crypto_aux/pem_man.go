// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
package crypto_aux

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	b0gus_assets "b0gus/assets"
	b0gus_config "b0gus/configs"

	ssh "golang.org/x/crypto/ssh"
)

/*- Load ssh pem from given path
 * inParam: pem_path string; the path of the pem file, invisible to remote address
 * return ssh.Signer, error; if success, return ssh.Signer object and nil,
 * else return nil and error
 */
func loadSSHhostPem(pem_path string) (ssh.Signer, error) {
	pem_fd, err := os.Open(pem_path)
	if err != nil {
		open_pem_failure := b0gus_assets.GetLocalizedMsg(
			b0gus_config.GetLang(),
			"crypto_aux.PemFileOpenFailure",
			map[string]any{"PemPath": pem_path},
		)
		// "Failed to read PEM file for SSH host key!"
		b0gus_config.Logger.Error(open_pem_failure)
		return nil, err
	}
	pem_bytes, err := io.ReadAll(pem_fd)
	if err != nil {
		read_pem_failure := b0gus_assets.GetLocalizedMsg(
			b0gus_config.GetLang(),
			"crypto_aux.PemFileReadFailure", nil,
		)
		// "Failed to read PEM file for SSH host key!"
		b0gus_config.Logger.Error(read_pem_failure)
		return nil, err
	}
	pem_fd.Close()
	pem_block, _ := pem.Decode(pem_bytes)
	if pem_block == nil {
		pem_decode_failure := b0gus_assets.GetLocalizedMsg(
			b0gus_config.GetLang(),
			"crypto_aux.PemFileDecodeFailure", nil,
		)
		b0gus_config.Logger.Error(pem_decode_failure)
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
		return nil, errors.New("unsupported signing pem format")
	}
}

func handle_ed25519() (*ed25519.PrivateKey, error) {
	_, private_key, err := ed25519.GenerateKey(rand.Reader)
	return &private_key, err
}

func handle_elliptic(pem_len uint64) (*ecdsa.PrivateKey, error) {
	var choice elliptic.Curve
	switch pem_len {
	case 224:
		choice = elliptic.P224()
	case 256:
		choice = elliptic.P256()
	case 384:
		choice = elliptic.P384()
	case 521:
		choice = elliptic.P521()
	default:
		invalid_len := b0gus_assets.GetLocalizedMsg(
			b0gus_config.GetLang(),
			"crypto_aux.InvalidLengthForEllipticCurve",
			map[string]any{
				"PemLen": pem_len,
			},
		)
		return nil, errors.New(invalid_len)
	}
	return ecdsa.GenerateKey(choice, rand.Reader)
}

func handle_rsa(pem_len uint64) (*rsa.PrivateKey, error) {
	switch pem_len {
	case 1024:
		// this is not secure at all. Use 2048 instead
		pem_len = 2048
	case 2048:
	case 4096:
	case 8192:
	default:
		invalid_len := b0gus_assets.GetLocalizedMsg(
			b0gus_config.GetLang(),
			"crypto_aux.InvalidLengthForRSA",
			map[string]any{
				"PemLen": pem_len,
			},
		)
		return nil, errors.New(invalid_len)
	}
	return rsa.GenerateKey(rand.Reader, int(pem_len&0xFFFF_FFFF))
}

// Generate a new host key when a PEM file given by configuration is invalid/corrupted
// or not exists
func createSSHpem(
	pem_path string,
	pem_type string,
	pem_len uint64,
) (ssh.Signer, error) {
	var (
		host_pem any = nil
		err      error
	)
	switch strings.ToLower(pem_type) {
	case "ed25519":
		_host_pem, err := handle_ed25519()
		if err == nil && _host_pem != nil {
			host_pem = *_host_pem
		}
	case "elliptic":
		host_pem, err = handle_elliptic(pem_len)
	case "rsa":
		host_pem, err = handle_rsa(pem_len)

	default:
		b0gus_config.Logger.Error(
			"Invalid pem type was provided, won't generate any key!",
		)
		return nil, fmt.Errorf("invalid pem type:<%v>", pem_type)
	}
	if err != nil {
		b0gus_config.Logger.Error("Failed to generate host private key!")
		return nil, err
	}
	host_key, err := ssh.NewSignerFromKey(host_pem)
	if err != nil {
		b0gus_config.Logger.Error("Failed to set private key for ssh!")
		return nil, err
	}
	pem_file, err := os.Create(pem_path)
	if err != nil {
		b0gus_config.Logger.Errorf(
			"Failed to create pem file into path:%v!", pem_path,
		)
		return nil, err
	}
	defer pem_file.Close()
	host_pem_bytes, err := x509.MarshalPKCS8PrivateKey(host_pem)
	if err != nil {
		b0gus_config.Logger.Error("Failed to marshal pem bytes!")
		return nil, err
	}
	host_pem_block := pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: host_pem_bytes,
	}
	err = pem.Encode(pem_file, &host_pem_block)
	if err != nil {
		b0gus_config.Logger.Error("Failed to encode pem bytes into pem file!")
		return nil, err
	}
	return host_key, err
}

func LoadOrCreateSSHpem(
	pem_path string,
	pem_type string,
	pem_len uint64,
) ssh.Signer {
	pem_obj, err := loadSSHhostPem(pem_path)
	if err == nil {
		return pem_obj
	}
	var res ssh.Signer
	b0gus_config.Logger.Warnf(
		"Pem seems to be invalid or unsupported... detail:<%v>, b0gus will new one for you",
		err,
	)
	res, err = createSSHpem(pem_path, pem_type, pem_len)
	if err == nil {
		return res
	}
	b0gus_config.Logger.Errorf(
		"Unable to create pem at %s due to %v, won't execute start up server",
		pem_path, err,
	)
	return nil
}
