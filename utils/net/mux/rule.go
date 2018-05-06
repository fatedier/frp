package mux

type MatchFunc func(data []byte) (match bool)

var (
	HttpsNeedBytesNum uint32 = 1
	HttpNeedBytesNum  uint32 = 3
	YamuxNeedBytesNum uint32 = 2
)

var HttpsMatchFunc MatchFunc = func(data []byte) bool {
	if len(data) < int(HttpsNeedBytesNum) {
		return false
	}

	if data[0] == 0x16 {
		return true
	} else {
		return false
	}
}

// From https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods
var httpHeadBytes = map[string]struct{}{
	"GET": struct{}{},
	"HEA": struct{}{},
	"POS": struct{}{},
	"PUT": struct{}{},
	"DEL": struct{}{},
	"CON": struct{}{},
	"OPT": struct{}{},
	"TRA": struct{}{},
	"PAT": struct{}{},
}

var HttpMatchFunc MatchFunc = func(data []byte) bool {
	if len(data) < int(HttpNeedBytesNum) {
		return false
	}

	_, ok := httpHeadBytes[string(data[:3])]
	return ok
}

// From https://github.com/hashicorp/yamux/blob/master/spec.md
var YamuxMatchFunc MatchFunc = func(data []byte) bool {
	if len(data) < int(YamuxNeedBytesNum) {
		return false
	}

	if data[0] == 0 && data[1] >= 0x0 && data[1] <= 0x3 {
		return true
	}
	return false
}
