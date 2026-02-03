package activities

import (
	"agent/internal/auth"
	"agent/internal/config"

	"go.temporal.io/sdk/client"
)

// Activities holds all activity implementations for the agent
// Refactored to use dependency injection instead of full config
type Activities struct {
	Config         *config.Config
	Auth           auth.AuthService
	Hub            *config.HubConfig
	TemporalClient client.Client
}

// NewActivities creates a new Activities instance with required dependencies
func NewActivities(config *config.Config, service auth.AuthService, hubConfig config.HubConfig, temporalClient client.Client) *Activities {
	return &Activities{
		Config:         config,
		Auth:           service,
		Hub:            &hubConfig,
		TemporalClient: temporalClient,
	}
}
