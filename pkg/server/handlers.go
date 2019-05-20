package server

import (
	"ddns-updater/pkg/models"
	"fmt"
	"net"
)

func healthcheckHandler(recordsConfigs []models.RecordConfigType) error {
	for i := range recordsConfigs {
		if recordsConfigs[i].Status.GetCode() == models.FAIL {
			return fmt.Errorf("%s", recordsConfigs[i].String())
		}
		if recordsConfigs[i].Settings.NoDNSLookup {
			continue
		}
		lookupIPs, err := net.LookupIP(recordsConfigs[i].Settings.BuildDomainName())
		if err != nil {
			return err
		}
		historyIPs := recordsConfigs[i].History.GetIPs()
		if len(historyIPs) == 0 {
			return fmt.Errorf("no set IP address found")
		}
		latestIP := historyIPs[0]
		for _, lookupIP := range lookupIPs {
			if lookupIP.String() != latestIP {
				return fmt.Errorf(
					"lookup IP address of %s is %s, not %s",
					recordsConfigs[i].Settings.BuildDomainName(),
					lookupIP,
					latestIP,
				)
			}
		}
	}
	return nil
}
