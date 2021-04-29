// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kibana

import (
	"fmt"
)

// FleetConfig represents the configuration for Fleet Server when building the enrollment command
type FleetConfig struct {
	EnrollmentToken string
	FleetServerPort int
	FleetServerURI  string
	// server
	ServerPolicyID string
}

// NewFleetConfig builds a new configuration for the fleet agent, defaulting ES credentials, URI and port.
func NewFleetConfig(token string, fleetServerHost string) (*FleetConfig, error) {
	cfg := &FleetConfig{
		EnrollmentToken: token,
		FleetServerURI:  fleetServerHost,
		FleetServerPort: 8220,
	}

	return cfg, nil
}

// Flags bootstrap flags for fleet server
func (cfg FleetConfig) Flags() []string {
	return []string{"-e", "-v", "--force", "--insecure", "--enrollment-token=" + cfg.EnrollmentToken, "--url", fmt.Sprintf("http://%s:%d", cfg.FleetServerURI, cfg.FleetServerPort)}
}
