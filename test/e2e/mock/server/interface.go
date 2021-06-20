package server

type Server interface {
	Run() error
	Close() error
	BindAddr() string
	BindPort() int
}
