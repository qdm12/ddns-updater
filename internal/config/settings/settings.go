package settings

import (
	"fmt"

	"github.com/qdm12/ddns-updater/internal/resolver"
	"github.com/qdm12/ddns-updater/internal/shoutrrr"
	"github.com/qdm12/gotree"
)

type Settings struct {
	Client   Client
	Update   Update
	PubIP    PubIP
	Resolver resolver.Settings
	IPv6     IPv6
	Server   Server
	Health   Health
	Paths    Paths
	Backup   Backup
	Logger   Logger
	Shoutrrr shoutrrr.Settings
}

func (s *Settings) SetDefaults() {
	s.Client.setDefaults()
	s.Update.setDefaults()
	s.PubIP.setDefaults()
	s.Resolver.SetDefaults()
	s.IPv6.setDefaults()
	s.Server.setDefaults()
	s.Health.SetDefaults()
	s.Paths.setDefaults()
	s.Backup.setDefaults()
	s.Logger.setDefaults()
	s.Shoutrrr.SetDefaults()
}

func (s Settings) MergeWith(other Settings) (merged Settings) {
	merged.Client = s.Client.mergeWith(other.Client)
	merged.Update = s.Update.mergeWith(other.Update)
	merged.PubIP = s.PubIP.mergeWith(other.PubIP)
	merged.Resolver = s.Resolver.MergeWith(other.Resolver)
	merged.IPv6 = s.IPv6.mergeWith(other.IPv6)
	merged.Server = s.Server.mergeWith(other.Server)
	merged.Health = s.Health.mergeWith(other.Health)
	merged.Paths = s.Paths.mergeWith(other.Paths)
	merged.Backup = s.Backup.mergeWith(other.Backup)
	merged.Logger = s.Logger.mergeWith(other.Logger)
	merged.Shoutrrr = s.Shoutrrr.MergeWith(other.Shoutrrr)
	return merged
}

func (s Settings) Validate() (err error) {
	type validator interface {
		Validate() (err error)
	}
	toValidate := map[string]validator{
		"client":    &s.Client,
		"update":    &s.Update,
		"public ip": &s.PubIP,
		"resolver":  &s.Resolver,
		"ipv6":      &s.IPv6,
		"server":    &s.Server,
		"health":    &s.Health,
		"paths":     &s.Paths,
		"backup":    &s.Backup,
		"logger":    &s.Logger,
		"shoutrrr":  &s.Shoutrrr,
	}

	for name, v := range toValidate {
		err = v.Validate()
		if err != nil {
			return fmt.Errorf("%s settings: %w", name, err)
		}
	}

	return nil
}

func (s Settings) String() string {
	return s.toLinesNode().String()
}

func (s Settings) toLinesNode() *gotree.Node {
	node := gotree.New("Settings summary:")
	node.AppendNode(s.Client.toLinesNode())
	node.AppendNode(s.Update.toLinesNode())
	node.AppendNode(s.PubIP.toLinesNode())
	node.AppendNode(s.Resolver.ToLinesNode())
	node.AppendNode(s.IPv6.toLinesNode())
	node.AppendNode(s.Server.toLinesNode())
	node.AppendNode(s.Health.toLinesNode())
	node.AppendNode(s.Paths.toLinesNode())
	node.AppendNode(s.Backup.toLinesNode())
	node.AppendNode(s.Logger.toLinesNode())
	node.AppendNode(s.Shoutrrr.ToLinesNode())
	return node
}
