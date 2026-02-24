// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
package crypto_aux

import "encoding/base64"

// turn bytes series of public key into base64 string
func PubKeyDeserialize(pub_byte []byte) string {
	return base64.StdEncoding.EncodeToString(pub_byte)
}

// turn base64 string of public key into bytes and attach an error if any
func PubKeySerialize(pub_str string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(pub_str)
}
