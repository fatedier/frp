package upnp

import (
	"context"

	"errors"
	"github.com/fatedier/frp/pkg/util/xlog"
	"golang.org/x/sync/errgroup"
	"net"
	"net/netip"
	"time"

	"github.com/huin/goupnp/dcps/internetgateway2"
	"github.com/jackpal/gateway"
)

const DEFAULT_UPNP_PROGRAM_DESCRIPTION = "helper-port-mapping"

type RouterClient interface {
	AddPortMapping(
		NewRemoteHost string,
		NewExternalPort uint16,
		NewProtocol string,
		NewInternalPort uint16,
		NewInternalClient string,
		NewEnabled bool,
		NewPortMappingDescription string,
		NewLeaseDuration uint32,
	) (err error)

	GetExternalIPAddress() (
		NewExternalIPAddress string,
		err error,
	)
}

func PickRouterClient(ctx context.Context) (RouterClient, error) {
	tasks, _ := errgroup.WithContext(ctx)
	// Request each type of client in parallel, and return what is found.
	var ip1Clients []*internetgateway2.WANIPConnection1
	tasks.Go(func() error {
		var err error
		ip1Clients, _, err = internetgateway2.NewWANIPConnection1Clients()
		return err
	})
	var ip2Clients []*internetgateway2.WANIPConnection2
	tasks.Go(func() error {
		var err error
		ip2Clients, _, err = internetgateway2.NewWANIPConnection2Clients()
		return err
	})
	var ppp1Clients []*internetgateway2.WANPPPConnection1
	tasks.Go(func() error {
		var err error
		ppp1Clients, _, err = internetgateway2.NewWANPPPConnection1Clients()
		return err
	})

	if err := tasks.Wait(); err != nil {
		return nil, err
	}

	// Trivial handling for where we find exactly one device to talk to, you
	// might want to provide more flexible handling than this if multiple
	// devices are found.
	switch {
	case len(ip2Clients) == 1:
		return ip2Clients[0], nil
	case len(ip1Clients) == 1:
		return ip1Clients[0], nil
	case len(ppp1Clients) == 1:
		return ppp1Clients[0], nil
	default:
		return nil, errors.New("multiple or no services found")
	}
}

func UPNP_ForwardPort(ctx context.Context,
	NewRemoteHost string,
	NewExternalPort uint16,
	NewProtocol string,
	NewInternalPort uint16,
	NewInternalClient string,
	NewPortMappingDescription string,
	NewLeaseDuration uint32,
) error {
	client, err := PickRouterClient(ctx)
	if err != nil {
		return err
	}

	return client.AddPortMapping(
		NewRemoteHost,
		// External port number to expose to Internet:
		NewExternalPort,
		// Forward TCP (this could be "UDP" if we wanted that instead).
		NewProtocol,
		// Internal port number on the LAN to forward to.
		// Some routers might not support this being different to the external
		// port number.
		NewInternalPort,
		// Internal address on the LAN we want to forward to.
		NewInternalClient,
		// Enabled:
		true,
		// Informational description for the client requesting the port forwarding.
		NewPortMappingDescription,
		// How long should the port forward last for in seconds.
		// If you want to keep it open for longer and potentially across router
		// resets, you might want to periodically request before this elapses.
		NewLeaseDuration,
	)
}

func AskForMapping(xl *xlog.Logger, remoteGetAddrs []string, localIps []string, localAddr net.Addr, description string) {

	xl.Tracef("makeRouterToNatThisHole: %v, localIps %v, localAddr=%v", remoteGetAddrs, localIps, localAddr.String())

	targetAddr := remoteGetAddrs[0]
	remoteAddrPort, err := netip.ParseAddrPort(targetAddr)
	if err != nil {
		xl.Errorf("netip.ParseAddrPort error: %v. parse: %v", err, targetAddr)
		return
	}

	localAddrStr := localAddr.String()
	localAddrPort, err := netip.ParseAddrPort(localAddrStr)
	if err != nil {
		xl.Errorf("netip.ParseAddrPort local error: %v. parse: %v", err, localAddrStr)

		return
	}

	targetForwardTo := ""
	if len(localIps) == 1 {
		targetForwardTo = localIps[0]
	} else {
		targetForwardToIp, err := gateway.DiscoverInterface()
		if err != nil {
			xl.Warnf("load Default interface error:%v", err)
		} else {
			targetForwardTo = targetForwardToIp.String()
		}
	}

	if targetForwardTo == "" && len(localIps) > 1 {
		targetForwardTo = localIps[0]
	}

	ctx, _ := context.WithTimeout(context.Background(), 50*time.Millisecond)

	xl.Infof("UPNP_ForwardPort: remoteAddrPort=%v, localAddrPort=%v, targetForwardToLocal=%v", remoteAddrPort, localAddrPort, targetForwardTo)
	err = UPNP_ForwardPort(
		ctx,
		/*NewRemoteHost*/ remoteAddrPort.Addr().String(),
		/*NewExternalPort*/ remoteAddrPort.Port(),
		/*NewProtocol*/ "UDP",

		/*NewInternalPort*/
		localAddrPort.Port(),
		/*NewInternalClient*/ targetForwardTo,
		/*NewPortMappingDescription*/ description,
		/*NewLeaseDuration*/ 360,
	)
	if err != nil {
		xl.Warnf("UPNP_ForwardPort error: %v.", err)

		return
	}

	xl.Tracef("UPNP_ForwardPort done")

}
