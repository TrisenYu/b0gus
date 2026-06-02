package crypto_aux

// SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha3"
	"crypto/sha512"
	"strings"

	"github.com/emmansun/gmsm/sm3"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/blake2s"
)

func Blake2sOnceDigest(key, payload []byte) []byte {
	d := blake2s.Sum256(append(key, payload...))
	return d[:]
}

func Blake256OnceDigest(key, payload []byte) []byte {
	d := blake2b.Sum256(append(key, payload...))
	return d[:]
}

func Blake384OnceDigest(key, payload []byte) []byte {
	d := blake2b.Sum384(append(key, payload...))
	return d[:]
}

func Blake512OnceDigest(key, payload []byte) []byte {
	d := blake2b.Sum512(append(key, payload...))
	return d[:]
}

func SM3OnceDigest(key, payload []byte) []byte {
	d := sm3.Sum(append(key, payload...))
	return d[:]
}

func MD5OnceDigest(key, payload []byte) []byte {
	d := md5.Sum(append(key, payload...))
	return d[:]
}

func SHA1OnceDigest(key, payload []byte) []byte {
	d := sha1.Sum(append(key, payload...))
	return d[:]
}

func SHA224OnceDigest(key, payload []byte) []byte {
	d := sha256.Sum224(append(key, payload...))
	return d[:]
}

func SHA256OnceDigest(key, payload []byte) []byte {
	d := sha256.Sum256(append(key, payload...))
	return d[:]
}

func SHA384OnceDigest(key, payload []byte) []byte {
	d := sha512.Sum384(append(key, payload...))
	return d[:]
}

func SHA512OnceDigest(key, payload []byte) []byte {
	d := sha512.Sum512(append(key, payload...))
	return d[:]
}

func SHA3Len256OnceDigest(key, payload []byte) []byte {
	d := sha3.New256().Sum(append(key, payload...))
	return d[:]
}

func SHA3Len384OnceDigest(key, payload []byte) []byte {
	d := sha3.New384().Sum(append(key, payload...))
	return d[:]
}

func SHA3Len512OnceDigest(key, payload []byte) []byte {
	d := sha3.New512().Sum(append(key, payload...))
	return d[:]
}

type OnceHashFnWithOptPadding func(first []byte, second []byte) []byte

func OnceHashByChoice(hashName string) OnceHashFnWithOptPadding {
	switch strings.ToLower(hashName) {
	case "blake256":
		return Blake256OnceDigest
	case "blake2s":
		return Blake2sOnceDigest
	case "blake384":
		return Blake384OnceDigest
	case "blake512":
		return Blake512OnceDigest
	case "md5": // not recommended.
		return MD5OnceDigest
	case "sha1":
		return SHA1OnceDigest
	case "sha224":
		return SHA224OnceDigest
	case "sha256":
		return SHA256OnceDigest
	case "sha384":
		return SHA384OnceDigest
	case "sha512":
		return SHA512OnceDigest
	case "sha3-256":

		return SHA3Len256OnceDigest
	case "sha3-384":

		return SHA3Len384OnceDigest
	case "sha3-512":
		return SHA3Len512OnceDigest
	case "sm3":
		return SM3OnceDigest
	default: // at least not nil.
		return SHA256OnceDigest
	}
}
