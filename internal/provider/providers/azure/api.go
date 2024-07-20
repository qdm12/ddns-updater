package azure

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"
)

func (p *Provider) createClient() (client *armdns.RecordSetsClient, err error) {
	credential, err := azidentity.NewClientSecretCredential(p.tenantID, p.clientID, p.clientSecret, nil)
	if err != nil {
		return nil, fmt.Errorf("creating client secret credential: %w", err)
	}

	client, err = armdns.NewRecordSetsClient(p.subscriptionID, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("creating record sets client: %w", err)
	}

	return client, nil
}

func (p *Provider) getRecordSet(ctx context.Context, client *armdns.RecordSetsClient,
	recordType armdns.RecordType) (response armdns.RecordSetsClientGetResponse, err error) {
	return client.Get(ctx, p.resourceGroupName, p.domain, p.owner, recordType, nil)
}

func (p *Provider) createRecordSet(ctx context.Context, client *armdns.RecordSetsClient,
	ip netip.Addr) (err error) {
	rrSet := armdns.RecordSet{Properties: &armdns.RecordSetProperties{}}
	recordType := armdns.RecordTypeA
	if ip.Is4() {
		rrSet.Properties.ARecords = []*armdns.ARecord{{IPv4Address: ptrTo(ip.String())}}
	} else {
		recordType = armdns.RecordTypeAAAA
		rrSet.Properties.AaaaRecords = []*armdns.AaaaRecord{{IPv6Address: ptrTo(ip.String())}}
	}
	_, err = client.CreateOrUpdate(ctx, p.resourceGroupName, p.domain,
		p.owner, recordType, rrSet, nil)
	if err != nil {
		return fmt.Errorf("creating record set: %w", err)
	}
	return nil
}

func (p *Provider) updateRecordSet(ctx context.Context, client *armdns.RecordSetsClient,
	response armdns.RecordSetsClientGetResponse, ip netip.Addr) (err error) {
	properties := response.Properties
	recordType := armdns.RecordTypeA
	if ip.Is4() {
		if len(properties.ARecords) == 0 {
			properties.ARecords = make([]*armdns.ARecord, 1)
		}
		for i := range properties.ARecords {
			properties.ARecords[i].IPv4Address = ptrTo(ip.String())
		}
	} else {
		recordType = armdns.RecordTypeAAAA
		if len(properties.AaaaRecords) == 0 {
			properties.AaaaRecords = make([]*armdns.AaaaRecord, 1)
		}
		for i := range properties.AaaaRecords {
			properties.AaaaRecords[i].IPv6Address = ptrTo(ip.String())
		}
	}
	rrSet := armdns.RecordSet{
		Etag:       response.Etag,
		Properties: properties,
	}

	_, err = client.CreateOrUpdate(ctx, p.resourceGroupName, p.domain,
		p.owner, recordType, rrSet, nil)
	return err
}
