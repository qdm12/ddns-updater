package doh

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Client_LookupNetIP(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, contentTypeDNSMessage, r.Header.Get("Content-Type"))

		query := new(dns.Msg)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, query.Unpack(body))

		require.Len(t, query.Question, 1)
		question := query.Question[0]

		response := new(dns.Msg)
		response.SetReply(query)
		if question.Name == "known.example.com." {
			switch question.Qtype {
			case dns.TypeA:
				record, err := dns.NewRR(question.Name + " 60 IN A 1.2.3.4")
				require.NoError(t, err)
				response.Answer = append(response.Answer, record)
			case dns.TypeAAAA:
				record, err := dns.NewRR(question.Name + " 60 IN AAAA 2001:db8::1")
				require.NoError(t, err)
				response.Answer = append(response.Answer, record)
			}
		}

		packed, err := response.Pack()
		require.NoError(t, err)
		w.Header().Set("Content-Type", contentTypeDNSMessage)
		_, _ = w.Write(packed)
	}))
	defer server.Close()

	client, err := New(server.URL, 5*time.Second, WithHTTPClient(server.Client()))
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("IPv4", func(t *testing.T) {
		addresses, err := client.LookupNetIP(ctx, "ip4", "known.example.com")
		require.NoError(t, err)
		require.Len(t, addresses, 1)
		assert.Equal(t, "1.2.3.4", addresses[0].String())
	})

	t.Run("IPv6", func(t *testing.T) {
		addresses, err := client.LookupNetIP(ctx, "ip6", "known.example.com")
		require.NoError(t, err)
		require.Len(t, addresses, 1)
		assert.Equal(t, "2001:db8::1", addresses[0].String())
	})

	t.Run("both", func(t *testing.T) {
		addresses, err := client.LookupNetIP(ctx, "ip", "known.example.com")
		require.NoError(t, err)
		assert.Len(t, addresses, 2)
	})

	t.Run("unknown name", func(t *testing.T) {
		_, err := client.LookupNetIP(ctx, "ip4", "unknown.example.com")
		require.Error(t, err)
		// The error message must end with "no such host" so that
		// callers can detect a name which does not exist.
		assert.Contains(t, err.Error(), "no such host")
	})

	t.Run("IP address given", func(t *testing.T) {
		addresses, err := client.LookupNetIP(ctx, "ip", "1.2.3.4")
		require.NoError(t, err)
		require.Len(t, addresses, 1)
		assert.Equal(t, "1.2.3.4", addresses[0].String())
	})

	t.Run("invalid network", func(t *testing.T) {
		_, err := client.LookupNetIP(ctx, "udp", "known.example.com")
		require.Error(t, err)
	})
}

func Test_New(t *testing.T) {
	t.Parallel()

	_, err := New("", time.Second)
	require.Error(t, err)

	_, err = New("http://cloudflare-dns.com/dns-query", time.Second)
	require.Error(t, err)

	client, err := New("https://cloudflare-dns.com/dns-query", time.Second)
	require.NoError(t, err)
	assert.NotNil(t, client)
}
