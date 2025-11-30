// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
package crypto_aux

import (
	"encoding/base64"
)

func PubKeyDeserialize(pub_byte []byte) string {
	return base64.StdEncoding.EncodeToString(pub_byte)
}

func PubKeySerialize(pub_str string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(pub_str)
}
