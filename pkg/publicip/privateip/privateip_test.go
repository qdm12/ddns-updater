package privateip

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errInjected = errors.New("injected")

// fakeInterface is an interface and its addresses, served by name.
type fakeInterface struct {
	iface    net.Interface
	addrs    []net.Addr
	addrsErr error
}

// fakeRetriever serves interfaces and addresses keyed by name and
// records the names for which Addrs was called.
type fakeRetriever struct {
	interfaces    []fakeInterface
	interfacesErr error
	addrsCalls    []string
	listCalls     int
}

func (f *fakeRetriever) Interfaces() ([]net.Interface, error) {
	f.listCalls++
	if f.interfacesErr != nil {
		return nil, f.interfacesErr
	}
	ifaces := make([]net.Interface, len(f.interfaces))
	for i, fi := range f.interfaces {
		ifaces[i] = fi.iface
	}
	return ifaces, nil
}

func (f *fakeRetriever) Addrs(iface net.Interface) ([]net.Addr, error) {
	f.addrsCalls = append(f.addrsCalls, iface.Name)
	for _, fi := range f.interfaces {
		if fi.iface.Name == iface.Name && fi.iface.Index == iface.Index {
			return fi.addrs, fi.addrsErr
		}
	}
	return nil, errInjected
}

func up(index int, name string) net.Interface {
	return net.Interface{Index: index, Name: name, Flags: net.FlagUp}
}

// mustNet returns the address of cidr (16-byte form for IPv4) with its network mask.
func mustNet(cidr string) *net.IPNet {
	ip, n, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}
	n.IP = ip
	return n
}

// mustNet4 is like mustNet with the 4-byte form for IPv4.
func mustNet4(cidr string) *net.IPNet {
	n := mustNet(cidr)
	n.IP = n.IP.To4()
	return n
}

type unsupportedAddr struct{}

func (unsupportedAddr) Network() string { return "unsupported" }
func (unsupportedAddr) String() string  { return "unsupported" }

func newFetcher(t *testing.T, name string, retriever InterfaceRetriever) *Fetcher {
	t.Helper()
	fetcher, err := New(Settings{Enabled: true, Interface: name}, retriever)
	require.NoError(t, err)
	return fetcher
}

func Test_Fetcher_selection(t *testing.T) {
	t.Parallel()

	bridge := func() fakeInterface {
		return fakeInterface{iface: up(1, "br-test"), addrs: []net.Addr{mustNet("10.128.11.1/16")}}
	}
	eth := func(index int, name string, addrs ...net.Addr) fakeInterface {
		return fakeInterface{iface: up(index, name), addrs: addrs}
	}

	testCases := map[string]struct {
		configured    string
		interfaces    func() []fakeInterface
		interfacesErr error
		wantIP        string // empty means failure
		wantErr       error
		wantCalls     []string
	}{
		"docker bridge before ethernet": {
			configured: "enp2s0",
			interfaces: func() []fakeInterface {
				return []fakeInterface{bridge(), eth(2, "enp2s0", mustNet("192.168.50.159/24"))}
			},
			wantIP:    "192.168.50.159",
			wantCalls: []string{"enp2s0"},
		},
		"interface list reordered": {
			configured: "enp2s0",
			interfaces: func() []fakeInterface {
				return []fakeInterface{eth(2, "enp2s0", mustNet("192.168.50.159/24")), bridge()}
			},
			wantIP:    "192.168.50.159",
			wantCalls: []string{"enp2s0"},
		},
		"target missing with docker present": {
			configured: "enp2s0",
			interfaces: func() []fakeInterface { return []fakeInterface{bridge()} },
			wantErr:    ErrInterfaceNotFound,
		},
		"target down with docker present": {
			configured: "enp2s0",
			interfaces: func() []fakeInterface {
				down := fakeInterface{
					iface: net.Interface{Index: 2, Name: "enp2s0"},
					addrs: []net.Addr{mustNet("192.168.50.159/24")},
				}
				return []fakeInterface{bridge(), down}
			},
			wantErr: ErrInterfaceDown,
		},
		"target loopback with private alias": {
			configured: "lo",
			interfaces: func() []fakeInterface {
				lo := fakeInterface{
					iface: net.Interface{Index: 1, Name: "lo", Flags: net.FlagUp | net.FlagLoopback},
					addrs: []net.Addr{mustNet("192.168.50.159/24")},
				}
				return []fakeInterface{lo}
			},
			wantErr: ErrInterfaceLoopback,
		},
		"only ipv6 link-local and public addresses": {
			configured: "enp2s0",
			interfaces: func() []fakeInterface {
				return []fakeInterface{bridge(), eth(2, "enp2s0",
					mustNet("fe80::1/64"), mustNet("2001:db8::1/64"), mustNet("169.254.1.1/16"),
					mustNet("203.0.113.7/24"), mustNet("fd00::1/64"))}
			},
			wantErr:   ErrNoPrivateIPv4,
			wantCalls: []string{"enp2s0"},
		},
		"one private among public and ipv6": {
			configured: "enp2s0",
			interfaces: func() []fakeInterface {
				return []fakeInterface{eth(2, "enp2s0",
					mustNet("2001:db8::1/64"), mustNet("203.0.113.7/24"), mustNet("192.168.50.159/24"))}
			},
			wantIP:    "192.168.50.159",
			wantCalls: []string{"enp2s0"},
		},
		"two private addresses first order": {
			configured: "enp2s0",
			interfaces: func() []fakeInterface {
				return []fakeInterface{eth(2, "enp2s0", mustNet("192.168.50.159/24"), mustNet("10.0.0.5/8"))}
			},
			wantErr:   ErrAmbiguousPrivateIPv4,
			wantCalls: []string{"enp2s0"},
		},
		"two private addresses second order": {
			configured: "enp2s0",
			interfaces: func() []fakeInterface {
				return []fakeInterface{eth(2, "enp2s0", mustNet("10.0.0.5/8"), mustNet("192.168.50.159/24"))}
			},
			wantErr:   ErrAmbiguousPrivateIPv4,
			wantCalls: []string{"enp2s0"},
		},
		"duplicate mapped and ipaddr representations": {
			configured: "enp2s0",
			interfaces: func() []fakeInterface {
				return []fakeInterface{eth(2, "enp2s0",
					mustNet("192.168.50.159/24"),                                // 16-byte IPNet
					mustNet4("192.168.50.159/24"),                               // 4-byte IPNet
					&net.IPAddr{IP: net.ParseIP("192.168.50.159")},              // IPAddr
					&net.IPNet{IP: net.ParseIP("::ffff:192.168.50.159").To16()}, // mapped
				)}
			},
			wantIP:    "192.168.50.159",
			wantCalls: []string{"enp2s0"},
		},
		"case sensitive name": {
			configured: "Enp2s0",
			interfaces: func() []fakeInterface {
				return []fakeInterface{eth(2, "enp2s0", mustNet("192.168.50.159/24"))}
			},
			wantErr: ErrInterfaceNotFound,
		},
		"malformed entries alone": {
			configured: "enp2s0",
			interfaces: func() []fakeInterface {
				var nilNet *net.IPNet
				var nilAddr *net.IPAddr
				return []fakeInterface{eth(2, "enp2s0", nil, unsupportedAddr{}, nilNet, nilAddr,
					&net.IPNet{}, &net.IPNet{IP: net.IP{1, 2, 3}}, &net.IPNet{IP: net.IP{1, 2, 3, 4, 5}},
					&net.IPAddr{})}
			},
			wantErr:   ErrNoPrivateIPv4,
			wantCalls: []string{"enp2s0"},
		},
		"malformed entries beside one private": {
			configured: "enp2s0",
			interfaces: func() []fakeInterface {
				return []fakeInterface{eth(2, "enp2s0", nil, unsupportedAddr{},
					&net.IPNet{}, &net.IPNet{IP: net.IP{1, 2, 3}}, &net.IPNet{IP: net.IP{1, 2, 3, 4, 5}},
					mustNet("192.168.50.159/24"))}
			},
			wantIP:    "192.168.50.159",
			wantCalls: []string{"enp2s0"},
		},
		"interface list retrieval fails": {
			configured:    "enp2s0",
			interfacesErr: errInjected,
			interfaces:    func() []fakeInterface { return nil },
			wantErr:       errInjected,
		},
		"address retrieval fails and unrelated are not queried": {
			configured: "enp2s0",
			interfaces: func() []fakeInterface {
				broken := fakeInterface{iface: up(2, "enp2s0"), addrsErr: errInjected}
				return []fakeInterface{bridge(), broken}
			},
			wantErr:   errInjected,
			wantCalls: []string{"enp2s0"},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			methods := map[string]func(*Fetcher, context.Context) (netip.Addr, error){
				"IP":  (*Fetcher).IP,
				"IP4": (*Fetcher).IP4,
			}
			for methodName, method := range methods {
				t.Run(methodName, func(t *testing.T) {
					t.Parallel()

					retriever := &fakeRetriever{
						interfaces:    testCase.interfaces(),
						interfacesErr: testCase.interfacesErr,
					}
					fetcher := newFetcher(t, testCase.configured, retriever)

					ip, err := method(fetcher, t.Context())

					if testCase.wantIP != "" {
						require.NoError(t, err)
						assert.Equal(t, netip.MustParseAddr(testCase.wantIP), ip)
					} else {
						require.ErrorIs(t, err, testCase.wantErr)
						assert.False(t, ip.IsValid())
						assert.Contains(t, err.Error(), testCase.configured)
					}
					assert.Equal(t, testCase.wantCalls, retriever.addrsCalls)
				})
			}
		})
	}
}

func Test_Fetcher_tracksChanges(t *testing.T) {
	t.Parallel()

	retriever := &fakeRetriever{interfaces: []fakeInterface{
		{iface: up(1, "br-test"), addrs: []net.Addr{mustNet("10.128.11.1/16")}},
		{iface: up(2, "enp2s0"), addrs: []net.Addr{mustNet("192.168.50.159/24")}},
	}}
	fetcher := newFetcher(t, "enp2s0", retriever)

	ip, err := fetcher.IP4(t.Context())
	require.NoError(t, err)
	assert.Equal(t, netip.MustParseAddr("192.168.50.159"), ip)

	// DHCP replaces the address.
	retriever.interfaces[1].addrs = []net.Addr{mustNet("192.168.50.160/24")}
	ip, err = fetcher.IP4(t.Context())
	require.NoError(t, err)
	assert.Equal(t, netip.MustParseAddr("192.168.50.160"), ip)

	// The interface disappears: no fallback to the bridge, no stale address.
	retriever.interfaces = retriever.interfaces[:1]
	ip, err = fetcher.IP4(t.Context())
	require.ErrorIs(t, err, ErrInterfaceNotFound)
	assert.False(t, ip.IsValid())

	// The interface is recreated with a new index.
	retriever.interfaces = append(retriever.interfaces,
		fakeInterface{iface: up(7, "enp2s0"), addrs: []net.Addr{mustNet("192.168.50.160/24")}})
	ip, err = fetcher.IP4(t.Context())
	require.NoError(t, err)
	assert.Equal(t, netip.MustParseAddr("192.168.50.160"), ip)
}

func Test_New(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		settings  Settings
		retriever InterfaceRetriever
		wantErr   error
	}{
		"disabled":              {Settings{Interface: "enp2s0"}, &fakeRetriever{}, ErrDisabled},
		"empty interface":       {Settings{Enabled: true}, &fakeRetriever{}, ErrInterfaceNotSet},
		"whitespace interface":  {Settings{Enabled: true, Interface: "   "}, &fakeRetriever{}, ErrInterfaceNotSet},
		"nil retriever":         {Settings{Enabled: true, Interface: "enp2s0"}, nil, ErrRetrieverNil},
		"absent interface name": {Settings{Enabled: true, Interface: "does-not-exist"}, &fakeRetriever{}, nil},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fetcher, err := New(testCase.settings, testCase.retriever)

			if testCase.wantErr != nil {
				require.ErrorIs(t, err, testCase.wantErr)
				assert.Nil(t, fetcher)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, fetcher)
		})
	}
}

func Test_Fetcher_IP6(t *testing.T) {
	t.Parallel()

	retriever := &fakeRetriever{interfaces: []fakeInterface{
		{iface: up(2, "enp2s0"), addrs: []net.Addr{mustNet("fd00::1/64")}},
	}}
	fetcher := newFetcher(t, "enp2s0", retriever)

	ip, err := fetcher.IP6(t.Context())

	require.ErrorIs(t, err, ErrIPv6NotSupported)
	assert.False(t, ip.IsValid())
	assert.Empty(t, retriever.addrsCalls)
	assert.Zero(t, retriever.listCalls)
}
