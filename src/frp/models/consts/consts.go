package consts

// server status
const (
	Idle = iota
	Working
)

// connection type
const (
	CtlConn = iota
	WorkConn
)

// msg from client to server
const (
	CSHeartBeatReq = 1
)

// msg from server to client
const (
	SCHeartBeatRes = 100
)
