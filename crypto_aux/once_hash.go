// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
package crypto_aux

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
)

func MD5Oncedigest(payload, key []byte) []byte {
	d := md5.Sum(append(key, payload...))
	return d[:]
}

func SHA1Oncedigest(payload, key []byte) []byte {
	d := sha1.Sum(append(key, payload...))
	return d[:]
}
func SHA224Oncedigest(payload, key []byte) []byte {
	d := sha256.Sum224(append(key, payload...))
	return d[:]
}

func SHA256Oncedigest(payload, key []byte) []byte {
	d := sha256.Sum256(append(key, payload...))
	return d[:]
}

func SHA384Oncedigest(payload, key []byte) []byte {
	d := sha512.Sum384(append(key, payload...))
	return d[:]
}

func SHA512Oncedigest(payload, key []byte) []byte {
	d := sha512.Sum512(append(key, payload...))
	return d[:]
}
