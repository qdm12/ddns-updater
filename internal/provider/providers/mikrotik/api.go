package mikrotik

import (
	"github.com/go-routeros/routeros" //nolint:misspell
)

type addressListItem struct {
	id      string
	list    string
	address string
}

func getAddressListItems(client *routeros.Client,
	addressList string) (items []addressListItem, err error) {
	reply, err := client.Run("/ip/firewall/address-list/print",
		"?disabled=false", "?list="+addressList)
	if err != nil {
		return nil, err
	}

	items = make([]addressListItem, 0, len(reply.Re))
	for _, re := range reply.Re {
		item := addressListItem{
			id:      re.Map[".id"],
			list:    re.Map["list"],
			address: re.Map["address"],
		}
		if item.id == "" || item.address == "" {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}
