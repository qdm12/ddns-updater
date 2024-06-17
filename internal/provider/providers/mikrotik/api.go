package mikrotik

type addressListItem struct {
	id      string
	list    string
	address string
}

func getAddressListItems(client *client,
	addressList string) (items []addressListItem, err error) {
	reply, err := client.Run("/ip/firewall/address-list/print",
		"?disabled=false", "?list="+addressList)
	if err != nil {
		return nil, err
	}

	items = make([]addressListItem, 0, len(reply.sentences))
	for _, re := range reply.sentences {
		item := addressListItem{
			id:      re.mapping[".id"],
			list:    re.mapping["list"],
			address: re.mapping["address"],
		}
		if item.id == "" || item.address == "" {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}
