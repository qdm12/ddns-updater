package models

import (
	"sync"
	"time"
)

// RecordConfigType contains all the information to update and display a DNS record
type RecordConfigType struct { // internal
	Settings   SettingsType // fixed
	Status     statusType   // changes for each update
	History    historyType  // past information
	IsUpdating sync.Mutex   // just to wait for an update to finish on chQuit signaling
}

// NewRecordConfig returns a new recordConfig with settings
func NewRecordConfig(settings SettingsType, ips []string, tSuccess time.Time) RecordConfigType {
	return RecordConfigType{
		Settings: settings,
		History:  newHistory(ips, tSuccess),
	}
}

func (conf *RecordConfigType) String() string {
	return conf.Settings.String() + ": " + conf.Status.String() + "; " + conf.History.String()
}

func (conf *RecordConfigType) toHTML() HTMLRow {
	row := HTMLRow{
		Domain:   conf.Settings.getHTMLDomain(),
		Host:     conf.Settings.Host,
		Provider: conf.Settings.getHTMLProvider(),
		IPMethod: conf.Settings.getHTMLIPMethod(),
	}
	if conf.Status.code == UPTODATE {
		conf.Status.message = "No IP change for " + conf.History.GetTSuccessDuration()
	}
	row.Status = conf.Status.toHTML()
	ips := conf.History.GetIPs()
	latestIP := ips[0]
	if len(ips) > 0 {
		row.IP = "<a href=\"https://ipinfo.io/" + latestIP + "\">" + latestIP + "</a>"
	} else {
		row.IP = "N/A"
	}
	if len(ips) > 1 {
		for i, historicIP := range ips[1:] {
			if i != len(ips[1:])-1 { // not the last one
				historicIP += ", "
			}
			row.IPs = append(row.IPs, historicIP)
		}
	} else {
		row.IPs = []string{"N/A"}
	}
	return row
}
