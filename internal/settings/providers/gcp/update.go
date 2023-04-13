package gcp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/qdm12/ddns-updater/internal/settings/constants"
	ddnserrors "github.com/qdm12/ddns-updater/internal/settings/errors"
	clouddns "google.golang.org/api/dns/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

func (p *Provider) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	recordType := constants.A
	if ip.To4() == nil {
		recordType = constants.AAAA
	}

	ddnsService, err := clouddns.NewService(ctx,
		option.WithCredentialsJSON(p.credentials),
		option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("creating GCP DDNS service: %w", err)
	}
	rrSetsService := clouddns.NewResourceRecordSetsService(ddnsService)

	fqdn := fmt.Sprintf("%s.%s.", p.host, p.domain)

	recordResourceSet, err := p.getResourceRecordSet(rrSetsService, fqdn, recordType)
	rrSetFound := true
	if err != nil {
		if errors.Is(err, ddnserrors.ErrNotFound) {
			rrSetFound = false // not finding the record is fine
		} else {
			return nil, fmt.Errorf("getting record resource set: %w", err)
		}
	}

	for _, rrdata := range recordResourceSet.Rrdatas {
		if rrdata == ip.String() {
			// already up to date
			return ip, nil
		}
	}

	if !rrSetFound {
		err = p.createRecord(rrSetsService, fqdn, recordType, ip)
		if err != nil {
			return nil, fmt.Errorf("creating record: %w", err)
		}
		return ip, nil
	}

	err = p.updateRecord(rrSetsService, fqdn, recordType, ip)
	if err != nil {
		return nil, fmt.Errorf("updating record: %w", err)
	}

	return ip, nil
}

func (p *Provider) getResourceRecordSet(rrSetsService *clouddns.ResourceRecordSetsService,
	fqdn, recordType string) (resourceRecordSet *clouddns.ResourceRecordSet, err error) {
	call := rrSetsService.Get(p.project, p.zone, fqdn, recordType)
	resourceRecordSet, err = call.Do()
	if err != nil {
		googleAPIError := new(googleapi.Error)
		if errors.As(err, &googleAPIError) && googleAPIError.Code == http.StatusNotFound {
			return nil, fmt.Errorf("%w: %w", ddnserrors.ErrNotFound, err)
		}
		return nil, err
	}

	return resourceRecordSet, nil
}

func (p *Provider) createRecord(rrSetsService *clouddns.ResourceRecordSetsService,
	fqdn, recordType string, ip net.IP) (err error) {
	rrSet := &clouddns.ResourceRecordSet{
		Name:    fqdn,
		Rrdatas: []string{ip.String()},
		Type:    recordType,
	}
	rrSetCall := rrSetsService.Create(p.project, p.zone, rrSet)
	_, err = rrSetCall.Do()
	return err
}

func (p *Provider) updateRecord(rrSetsService *clouddns.ResourceRecordSetsService,
	fqdn, recordType string, ip net.IP) (err error) {
	rrSet := &clouddns.ResourceRecordSet{
		Name:    fqdn,
		Rrdatas: []string{ip.String()},
		Type:    recordType,
	}
	rrSetCall := rrSetsService.Patch(p.project, p.zone, fqdn, recordType, rrSet)
	_, err = rrSetCall.Do()
	return err
}
