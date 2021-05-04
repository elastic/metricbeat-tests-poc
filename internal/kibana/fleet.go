// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kibana

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// FleetConfig represents the configuration for Fleet Server when building the enrollment command
type FleetConfig struct {
	EnrollmentToken          string
	ElasticsearchPort        int
	ElasticsearchURI         string
	ElasticsearchCredentials string
	KibanaPort               int
	KibanaURI                string
	FleetServerPort          int
	FleetServerURI           string
<<<<<<< HEAD
	// server
	BootstrapFleetServer bool
	ServerPolicyID       string
}

// NewFleetConfig builds a new configuration for the fleet agent, defaulting ES credentials, URI and port.
// If the 'bootstrappFleetServer' flag is true, the it will create the config for the initial fleet server
// used to bootstrap Fleet Server
// If the 'fleetServerMode' flag is true, the it will create the config for an agent using an existing Fleet
// Server to connect to Fleet. It will also retrieve the default policy ID for fleet server
func NewFleetConfig(token string, bootstrapFleetServer bool, fleetServerMode bool) (*FleetConfig, error) {
=======
}

// NewFleetConfig builds a new configuration for the fleet agent, defaulting fleet-server host, ES credentials, URI and port.
func NewFleetConfig(token string) (*FleetConfig, error) {
>>>>>>> 82380ced... chore: remove unused code (#1119)
	cfg := &FleetConfig{
		EnrollmentToken:          token,
		ElasticsearchCredentials: "elastic:changeme",
		ElasticsearchPort:        9200,
		ElasticsearchURI:         "elasticsearch",
		KibanaPort:               5601,
		KibanaURI:                "kibana",
		FleetServerPort:          8220,
<<<<<<< HEAD
		FleetServerURI:           "localhost",
	}

	client, err := NewClient()
	if err != nil {
		return cfg, err
	}

	if fleetServerMode {
		defaultFleetServerPolicy, err := client.GetDefaultPolicy(true)
		if err != nil {
			return nil, err
		}

		cfg.ServerPolicyID = defaultFleetServerPolicy.ID

		log.WithFields(log.Fields{
			"elasticsearch":     cfg.ElasticsearchURI,
			"elasticsearchPort": cfg.ElasticsearchPort,
			"policyID":          cfg.ServerPolicyID,
			"token":             cfg.EnrollmentToken,
		}).Debug("Fleet Server config created")
	}
=======
		FleetServerURI:           "fleet-server",
	}

	log.WithFields(log.Fields{
		"elasticsearch":     cfg.ElasticsearchURI,
		"elasticsearchPort": cfg.ElasticsearchPort,
		"token":             cfg.EnrollmentToken,
	}).Debug("Fleet Server config created")
>>>>>>> 82380ced... chore: remove unused code (#1119)

	return cfg, nil
}

// Flags bootstrap flags for fleet server
func (cfg FleetConfig) Flags() []string {
<<<<<<< HEAD
	if cfg.BootstrapFleetServer {
		// TO-DO: remove all code to calculate the fleet-server policy, because it's inferred by the fleet-server
		return []string{
			"--force",
			"--fleet-server-es", fmt.Sprintf("http://%s@%s:%d", cfg.ElasticsearchCredentials, cfg.ElasticsearchURI, cfg.ElasticsearchPort),
		}
	}

=======
>>>>>>> 82380ced... chore: remove unused code (#1119)
	/*
		// agent using an already bootstrapped fleet-server
		fleetServerHost := "https://hostname_of_the_bootstrapped_fleet_server:8220"
		return []string{
			"-e", "-v", "--force", "--insecure",
			// ensure the enrollment belongs to the default policy
			"--enrollment-token=" + cfg.EnrollmentToken,
			"--url", fleetServerHost,
		}
	*/

<<<<<<< HEAD
	baseFlags := []string{"-e", "-v", "--force", "--insecure", "--enrollment-token=" + cfg.EnrollmentToken}
	if common.AgentVersionBase == "7.13.0-SNAPSHOT" {
		return append(baseFlags, "--url", fmt.Sprintf("http://%s@%s:%d", cfg.ElasticsearchCredentials, cfg.FleetServerURI, cfg.FleetServerPort))
	}

	if cfg.ServerPolicyID != "" {
		baseFlags = append(baseFlags, "--fleet-server-insecure-http", "--fleet-server", fmt.Sprintf("http://%s@%s:%d", cfg.ElasticsearchCredentials, cfg.ElasticsearchURI, cfg.ElasticsearchPort), "--fleet-server-host=http://0.0.0.0", "--fleet-server-policy", cfg.ServerPolicyID)
=======
	flags := []string{
		"-e", "-v", "--force", "--insecure", "--enrollment-token=" + cfg.EnrollmentToken,
		"--url", fmt.Sprintf("http://%s:%d", cfg.FleetServerURI, cfg.FleetServerPort),
>>>>>>> 82380ced... chore: remove unused code (#1119)
	}

	return flags
}
