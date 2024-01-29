package config

import (
	"fmt"

	"github.com/qdm12/gosettings/reader"
	"github.com/qdm12/gotree"
)

type Config struct {
	Client   Client
	Update   Update
	PubIP    PubIP
	Resolver Resolver
	Server   Server
	Health   Health
	Paths    Paths
	Backup   Backup
	Logger   Logger
	Shoutrrr Shoutrrr
}

func (c *Config) SetDefaults() {
	c.Client.setDefaults()
	c.Update.setDefaults()
	c.PubIP.setDefaults()
	c.Resolver.setDefaults()
	c.Server.setDefaults()
	c.Health.SetDefaults()
	c.Paths.setDefaults()
	c.Backup.setDefaults()
	c.Logger.setDefaults()
	c.Shoutrrr.setDefaults()
}

func (c Config) Validate() (err error) {
	type validator interface {
		Validate() (err error)
	}
	toValidate := map[string]validator{
		"client":    &c.Client,
		"update":    &c.Update,
		"public ip": &c.PubIP,
		"resolver":  &c.Resolver,
		"server":    &c.Server,
		"health":    &c.Health,
		"paths":     &c.Paths,
		"backup":    &c.Backup,
		"logger":    &c.Logger,
		"shoutrrr":  &c.Shoutrrr,
	}

	for name, v := range toValidate {
		err = v.Validate()
		if err != nil {
			return fmt.Errorf("%s settings: %w", name, err)
		}
	}

	return nil
}

func (c Config) String() string {
	return c.toLinesNode().String()
}

func (c Config) toLinesNode() *gotree.Node {
	node := gotree.New("Settings summary:")
	node.AppendNode(c.Client.toLinesNode())
	node.AppendNode(c.Update.toLinesNode())
	node.AppendNode(c.PubIP.toLinesNode())
	node.AppendNode(c.Resolver.ToLinesNode())
	node.AppendNode(c.Server.toLinesNode())
	node.AppendNode(c.Health.toLinesNode())
	node.AppendNode(c.Paths.toLinesNode())
	node.AppendNode(c.Backup.toLinesNode())
	node.AppendNode(c.Logger.toLinesNode())
	node.AppendNode(c.Shoutrrr.ToLinesNode())
	return node
}

func (c *Config) Read(reader *reader.Reader,
	warner Warner) (err error) {
	err = c.Client.read(reader)
	if err != nil {
		return fmt.Errorf("reading client settings: %w", err)
	}

	err = c.Update.read(reader, warner)
	if err != nil {
		return fmt.Errorf("reading update settings: %w", err)
	}

	err = c.PubIP.read(reader, warner)
	if err != nil {
		return fmt.Errorf("reading public IP settings: %w", err)
	}

	err = c.Resolver.read(reader)
	if err != nil {
		return fmt.Errorf("reading resolver settings: %w", err)
	}

	err = c.Server.read(reader, warner)
	if err != nil {
		return fmt.Errorf("reading server settings: %w", err)
	}

	c.Health.Read(reader)
	c.Paths.read(reader)

	err = c.Backup.read(reader)
	if err != nil {
		return fmt.Errorf("reading backup settings: %w", err)
	}

	c.Logger.read(reader)

	err = c.Shoutrrr.read(reader, warner)
	if err != nil {
		return fmt.Errorf("reading shoutrrr settings: %w", err)
	}

	return nil
}
