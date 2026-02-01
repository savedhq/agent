package activities

import (
	"agent/internal/auth"
	"agent/internal/config"
	"log"
	"os"
)

// Activities holds all activity implementations for the agent
// Refactored to use dependency injection instead of full config
type Activities struct {
	Config *config.Config
	Auth   auth.AuthService
	Hub    *config.HubConfig
	logger *log.Logger
}

// NewActivities creates a new Activities instance with required dependencies
func NewActivities(config *config.Config, service auth.AuthService, hubConfig config.HubConfig) *Activities {
	return &Activities{
		Config: config,
		Auth:   service,
		Hub:    &hubConfig,
		logger: log.New(os.Stdout, "activities: ", log.LstdFlags),
	}
}
