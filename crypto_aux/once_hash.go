package crypto_aux

// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"strings"

	"github.com/emmansun/gmsm/sm3"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/blake2s"
)

func Blake2sOnceDigest(payload, key []byte) []byte {
	d := blake2s.Sum256(append(key, payload...))
	return d[:]
}

func Blake256OnceDigest(payload, key []byte) []byte {
	d := blake2b.Sum256(append(key, payload...))
	return d[:]
}

func Blake384OnceDigest(payload, key []byte) []byte {
	d := blake2b.Sum384(append(key, payload...))
	return d[:]
}

func Blake512OnceDigest(payload, key []byte) []byte {
	d := blake2b.Sum512(append(key, payload...))
	return d[:]
}

func SM3OnceDigest(payload, key []byte) []byte {
	d := sm3.Sum(append(key, payload...))
	return d[:]
}

func MD5OnceDigest(payload, key []byte) []byte {
	d := md5.Sum(append(key, payload...))
	return d[:]
}

func SHA1OnceDigest(payload, key []byte) []byte {
	d := sha1.Sum(append(key, payload...))
	return d[:]
}
func SHA224OnceDigest(payload, key []byte) []byte {
	d := sha256.Sum224(append(key, payload...))
	return d[:]
}

func SHA256OnceDigest(payload, key []byte) []byte {
	d := sha256.Sum256(append(key, payload...))
	return d[:]
}

func SHA384OnceDigest(payload, key []byte) []byte {
	d := sha512.Sum384(append(key, payload...))
	return d[:]
}

func SHA512OnceDigest(payload, key []byte) []byte {
	d := sha512.Sum512(append(key, payload...))
	return d[:]
}

type OnceHashWithPadding func([]byte, []byte) []byte

func OnceHashByChoice(hashName string) OnceHashWithPadding {
	hashName = strings.ToLower(hashName)
	switch hashName {
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
	case "sm3":
		return SM3OnceDigest
	default: // at least not nil.
		return SHA256OnceDigest
	}
}
