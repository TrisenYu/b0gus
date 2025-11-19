package crypto_aux

import (
	"encoding/base64"
)

func PubKeyDeserialize(pub_byte []byte) string {
	return base64.StdEncoding.EncodeToString(pub_byte)
}
