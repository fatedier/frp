package consts

import "path/filepath"

var (
	FRPS_BIN_PATH = "../../bin/frps"
	FRPC_BIN_PATH = "../../bin/frpc"

	FRPS_SUB_BIN_PATH = "../../../bin/frps"
	FRPC_SUB_BIN_PATH = "../../../bin/frpc"

	FRPS_NORMAL_CONFIG = "./auto_test_frps.ini"
	FRPC_NORMAL_CONFIG = "./auto_test_frpc.ini"

	SERVER_ADDR = "127.0.0.1"
	ADMIN_ADDR  = "127.0.0.1:10600"
	ADMIN_USER  = "abc"
	ADMIN_PWD   = "abc"

	TEST_STR                    = "frp is a fast reverse proxy to help you expose a local server behind a NAT or firewall to the internet."
	TEST_TCP_PORT        int    = 10701
	TEST_TCP2_PORT       int    = 10702
	TEST_TCP_FRP_PORT    int    = 10801
	TEST_TCP2_FRP_PORT   int    = 10802
	TEST_TCP_EC_FRP_PORT int    = 10901
	TEST_TCP_ECHO_STR    string = "tcp type:" + TEST_STR

	TEST_UDP_PORT        int    = 10702
	TEST_UDP_FRP_PORT    int    = 10802
	TEST_UDP_EC_FRP_PORT int    = 10902
	TEST_UDP_ECHO_STR    string = "udp type:" + TEST_STR

	TEST_UNIX_DOMAIN_ADDR     string = "/tmp/frp_echo_server.sock"
	TEST_UNIX_DOMAIN_FRP_PORT int    = 10803
	TEST_UNIX_DOMAIN_STR      string = "unix domain type:" + TEST_STR

	TEST_HTTP_PORT       int    = 10704
	TEST_HTTP_FRP_PORT   int    = 10804
	TEST_HTTP_NORMAL_STR string = "http normal string: " + TEST_STR
	TEST_HTTP_FOO_STR    string = "http foo string: " + TEST_STR
	TEST_HTTP_BAR_STR    string = "http bar string: " + TEST_STR

	TEST_TCP_MUX_FRP_PORT int = 10806

	TEST_STCP_FRP_PORT    int    = 10805
	TEST_STCP_EC_FRP_PORT int    = 10905
	TEST_STCP_ECHO_STR    string = "stcp type:" + TEST_STR

	TEST_SUDP_FRP_PORT int    = 10816
	TEST_SUDP_ECHO_STR string = "sudp type:" + TEST_STR

	ProxyTcpPortNotAllowed  string = "tcp_port_not_allowed"
	ProxyTcpPortUnavailable string = "tcp_port_unavailable"
	ProxyTcpPortNormal      string = "tcp_port_normal"
	ProxyTcpRandomPort      string = "tcp_random_port"
	ProxyUdpPortNotAllowed  string = "udp_port_not_allowed"
	ProxyUdpPortNormal      string = "udp_port_normal"
	ProxyUdpRandomPort      string = "udp_random_port"
	ProxyHttpProxy          string = "http_proxy"

	ProxyRangeTcpPrefix string = "range_tcp"
)

func init() {
	if path, err := filepath.Abs(FRPS_BIN_PATH); err != nil {
		panic(err)
	} else {
		FRPS_BIN_PATH = path
	}

	if path, err := filepath.Abs(FRPC_BIN_PATH); err != nil {
		panic(err)
	} else {
		FRPC_BIN_PATH = path
	}
}
