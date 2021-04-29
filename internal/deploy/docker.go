// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package deploy

import (
	"context"
	"strings"

	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/compose"
	"github.com/elastic/e2e-testing/internal/docker"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
)

// DockerDeploymentManifest deploy manifest for docker
type dockerDeploymentManifest struct {
	ctx context.Context
}

// NewDockerDeployment initializes docker deployment
func NewDockerDeployment() Deployment {
	return &dockerDeploymentManifest{ctx: context.Background()}
}

// Add adds services deployment
func (c *dockerDeploymentManifest) Add(services []string, env map[string]string) error {
	serviceManager := compose.NewServiceManager()

	return serviceManager.AddServicesToCompose(c.ctx, services[0], services[1:], env)
}

// Bootstrap sets up environment with docker compose
func (c *dockerDeploymentManifest) Bootstrap() error {
	serviceManager := compose.NewServiceManager()
	common.ProfileEnv = map[string]string{
		"kibanaVersion": common.KibanaVersion,
		"stackVersion":  common.StackVersion,
	}

	common.ProfileEnv["kibanaDockerNamespace"] = "kibana"
	if strings.HasPrefix(common.KibanaVersion, "pr") || utils.IsCommit(common.KibanaVersion) {
		// because it comes from a PR
		common.ProfileEnv["kibanaDockerNamespace"] = "observability-ci"
	}

	profile := common.FleetProfileName
	err := serviceManager.RunCompose(c.ctx, true, []string{profile}, common.ProfileEnv)
	if err != nil {
		log.WithFields(log.Fields{
			"profile": profile,
			"error":   err.Error(),
		}).Fatal("Could not run the runtime dependencies for the profile.")
	}

	kibanaClient, err := kibana.NewClient()
	if err != nil {
		log.WithField("error", err).Fatal("Unable to create kibana client")
	}

	err = kibanaClient.WaitForFleet()
	if err != nil {
		log.WithField("error", err).Fatal("Fleet could not be initialized")
	}
	return nil
}

// Destroy teardown docker environment
func (c *dockerDeploymentManifest) Destroy() error {
	serviceManager := compose.NewServiceManager()
	err := serviceManager.StopCompose(c.ctx, true, []string{common.FleetProfileName})
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"profile": common.FleetProfileName,
		}).Fatal("Could not destroy the runtime dependencies for the profile.")
	}
	return nil
}

// Inspect inspects a service
func (c *dockerDeploymentManifest) Inspect(service string) (*ServiceManifest, error) {
	inspect, err := docker.InspectContainer(service)
	if err != nil {
		return &ServiceManifest{}, err
	}
	return &ServiceManifest{
		ID:         inspect.ID,
		Name:       strings.TrimPrefix(inspect.Name, "/"),
		Connection: service,
		Hostname:   inspect.NetworkSettings.Networks["fleet_default"].Aliases[0],
	}, nil
}

// Remove remove services from deployment
func (c *dockerDeploymentManifest) Remove(services []string, env map[string]string) error {
	serviceManager := compose.NewServiceManager()

	return serviceManager.RemoveServicesFromCompose(c.ctx, services[0], services[1:], env)
}
