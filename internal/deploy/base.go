// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package deploy

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Deployment interface for operations dealing with deployments of the bits
// required for testing
type Deployment interface {
	Add(services []string, env map[string]string) error // adds a service to deployment
	Bootstrap() error
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

// New creates a new deployment
func New(provider string) Deployment {
	if strings.EqualFold(provider, "docker") {
		log.Trace("Docker Deployment Selected")
		return newDockerDeploy()
	}
	if strings.EqualFold(provider, "kubernetes") {
		log.Trace("Kubernetes Deployment Selected")
		return newK8sDeploy()
	}
	return nil
}
