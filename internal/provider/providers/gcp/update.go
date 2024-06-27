package gcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/qdm12/ddns-updater/internal/provider/constants"
	ddnserrors "github.com/qdm12/ddns-updater/internal/provider/errors"
)

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	client, err = createOauth2Client(ctx, client, p.credentials)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating OAuth2 client: %w", err)
	}

	fqdn := fmt.Sprintf("%s.%s.", p.owner, p.domain)

	recordResourceSet, err := p.getRRSet(ctx, client, fqdn, recordType)
	rrSetFound := true
	if err != nil {
		if errors.Is(err, ddnserrors.ErrRecordResourceSetNotFound) {
			rrSetFound = false // not finding the record is fine
		} else {
			return netip.Addr{}, fmt.Errorf("getting record resource set: %w", err)
		}
	}

	for _, rrdata := range recordResourceSet.Rrdatas {
		if rrdata == ip.String() {
			// already up to date
			return ip, nil
		}
	}

	if !rrSetFound {
		err = p.createRRSet(ctx, client, fqdn, recordType, ip)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("creating record: %w", err)
		}
		return ip, nil
	}

	err = p.patchRRSet(ctx, client, fqdn, recordType, ip)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("updating record: %w", err)
	}

	return ip, nil
}
