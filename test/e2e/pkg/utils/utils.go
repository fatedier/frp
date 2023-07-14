package utils

import (
	"github.com/cristalhq/base64"
)

func BasicAuth(username, passwd string) string {
	auth := username + ":" + passwd
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}
