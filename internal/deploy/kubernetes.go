// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package deploy

import (
	"context"
	"strings"

	"github.com/elastic/e2e-testing/internal/compose"
	"github.com/elastic/e2e-testing/internal/docker"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/kubernetes"
	log "github.com/sirupsen/logrus"
)

var cluster kubernetes.Cluster
var kubectl kubernetes.Control

// KubernetesDeploymentManifest deploy manifest for kubernetes
type kubernetesDeploymentManifest struct {
	Context context.Context
}

func newK8sDeploy() Deployment {
	return &kubernetesDeploymentManifest{Context: context.Background()}
}

// Add adds services deployment
func (c *kubernetesDeploymentManifest) Add(services []string, env map[string]string) error {
	serviceManager := compose.NewServiceManager()

	return serviceManager.AddServicesToCompose(c.Context, services[0], services[1:], env)
}

// Bootstrap sets up environment with docker compose
func (c *kubernetesDeploymentManifest) Bootstrap() error {
	err := cluster.Initialize(c.Context, "../../../cli/config/kubernetes/kind.yaml")
	if err != nil {
		return err
	}

	kubectl = cluster.Kubectl().WithNamespace(c.Context, "default")
	_, err = kubectl.Run(c.Context, "apply", "-k", "../../../cli/config/kubernetes/base")
	if err != nil {
		return err
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
func (c *kubernetesDeploymentManifest) Destroy() error {
	kubectl = cluster.Kubectl().WithNamespace(c.Context, "")
	// err = cluster.Cleanup(c.Context)
	// if err != nil {
	// 	log.WithFields(log.Fields{
	// 		"error":   err,
	// 		"profile": common.FleetProfileName,
	// 	}).Fatal("Could not destroy the runtime dependencies for the profile.")
	// }
	return nil
}

// Inspect inspects a service
func (c *kubernetesDeploymentManifest) Inspect(service string) (*ServiceManifest, error) {
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
func (c *kubernetesDeploymentManifest) Remove(services []string, env map[string]string) error {
	serviceManager := compose.NewServiceManager()

	return serviceManager.RemoveServicesFromCompose(c.Context, services[0], services[1:], env)
}
