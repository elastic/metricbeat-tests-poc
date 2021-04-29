// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package deploy

import (
	"context"
	"strings"

	"github.com/elastic/e2e-testing/internal/common"
	log "github.com/sirupsen/logrus"
)

// Deployment interface for operations dealing with deployments of the bits
// required for testing
type Deployment interface {
	Add(services []string, env map[string]string) error    // adds a service to deployment
	Bootstrap() error                                      // Bootstraps an environment to test
	Destroy() error                                        // Teardown deployment
	Inspect(service string) (*ServiceManifest, error)      // inspects service
	Remove(services []string, env map[string]string) error // Removes services from deployment
}

// DeploymentManifest state management for deployment
type DeploymentManifest struct {
	Provider     string // deployment provider
	StackVersion string // elastic,kibana,fleet release version to deploy
	ctx          context.Context
}

// ServiceManifest information about a service in a deployment
type ServiceManifest struct {
	ID         string
	Name       string
	Connection string // a string representing how to connect to service
	Hostname   string
}

// NewClient loads deployment manifest for supported provider
func NewClient() Deployment {
	provider := strings.ToLower(common.Provider)
	switch provider {
	case "docker":
		return NewDockerDeployment()
	case "kubernetes":
		return NewKubernetesDeployment()
	default:
		log.WithField("provider", common.Provider).Fatal("Unknown deployment provider")
	}
	return nil
}
