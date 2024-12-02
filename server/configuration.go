package main

import (
	"encoding/json"
	"log"
	"reflect"

	"github.com/pkg/errors"
)

// configuration captures the plugin's external configuration as exposed in the Mattermost server
// configuration, as well as values computed from the configuration. Any public fields will be
// deserialized from the Mattermost server configuration in OnConfigurationChange.
//
// As plugins are inherently concurrent (hooks being called asynchronously), and the plugin
// configuration can change at any time, access to the configuration must be synchronized. The
// strategy used in this plugin is to guard a pointer to the configuration, and clone the entire
// struct whenever it changes. You may replace this with whatever strategy you choose.
//
// If you add non-reference types to your configuration struct, be sure to rewrite Clone as a deep
// copy appropriate for your types.

type ChannelAction struct {
	Action    string `json:"action"`
	ChannelID string `json:"channel_id"`
}

type Lead struct {
	Action    string `json:"action"`
	ChannelID string `json:"channel_id"`
}

type configuration struct {
	ChannelNewLead   string
	RabbitmqHost     string
	RabbitmqPort     string
	RabbitmqUser     string
	RabbitmqPassword string
	RabbitmqVhost    string
	AppHost          string
	BotToken         string
	channels         []ChannelAction
}

// Clone shallow copies the configuration. Your implementation may require a deep copy if
// your configuration has reference types.
func (c *configuration) Clone() *configuration {
	var clone = *c
	return &clone
}

// getConfiguration retrieves the active configuration under lock, making it safe to use
// concurrently. The active configuration may change underneath the client of this method, but
// the struct returned by this API call is considered immutable.
func (p *Plugin) getConfiguration() *configuration {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()

	if p.configuration == nil {
		return &configuration{}
	}

	return p.configuration
}

// setConfiguration replaces the active configuration under lock.
//
// Do not call setConfiguration while holding the configurationLock, as sync.Mutex is not
// reentrant. In particular, avoid using the plugin API entirely, as this may in turn trigger a
// hook back into the plugin. If that hook attempts to acquire this lock, a deadlock may occur.
//
// This method panics if setConfiguration is called with the existing configuration. This almost
// certainly means that the configuration was modified without being cloned and may result in
// an unsafe access.
func (p *Plugin) setConfiguration(configuration *configuration) {
	p.configurationLock.Lock()
	defer p.configurationLock.Unlock()

	if configuration != nil && p.configuration == configuration {
		// Ignore assignment if the configuration struct is empty. Go will optimize the
		// allocation for same to point at the same memory address, breaking the check
		// above.
		if reflect.ValueOf(*configuration).NumField() == 0 {
			return
		}

		panic("setConfiguration called with the existing configuration")
	}

	p.configuration = configuration
}

// OnConfigurationChange is invoked when configuration changes may have been made.
func (p *Plugin) OnConfigurationChange() error {
	var configuration = new(configuration)

	// Load the public configuration fields from the Mattermost server configuration.
	if err := p.API.LoadPluginConfiguration(configuration); err != nil {
		return errors.Wrap(err, "failed to load plugin configuration")
	}
	var channelIDs = configuration.ChannelNewLead
	var leads []Lead
	var channelActions []ChannelAction

	// Giải mã JSON vào mảng leads
	if err := json.Unmarshal([]byte(channelIDs), &leads); err != nil {
		log.Fatalf("Error unmarshaling JSON: %s", err)
	}
	teams, _ := p.API.GetTeams()
	for _, lead := range leads {
		log.Printf("Lead: %s", lead.Action)
		for _, team := range teams {
			log.Printf("Team: %s", team.Name)
			channel, err := p.API.GetChannelByNameForTeamName(team.Name, lead.ChannelID, false)
			if err != nil {
				break
			}
			log.Printf("Channel: %s", channel.Name)
			channelActions = append(channelActions, ChannelAction{
				Action:    lead.Action,
				ChannelID: channel.Id,
			})
		}
	}
	configuration.channels = channelActions

	p.setConfiguration(configuration)

	return nil
}
