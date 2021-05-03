// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/kubernetes"
	"github.com/pkg/errors"
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
	kubectl = cluster.Kubectl().WithNamespace(c.Context, "default")

	for _, service := range services {
		_, err := kubectl.Run(c.Context, "apply", "-k", fmt.Sprintf("../../../cli/config/kubernetes/overlays/%s", service))
		if err != nil {
			return err
		}
	}
	return nil
}

// Bootstrap sets up environment with kind
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

// Destroy teardown kubernetes environment
func (c *kubernetesDeploymentManifest) Destroy() error {
	kubectl = cluster.Kubectl().WithNamespace(c.Context, "default")
	cluster.Cleanup(c.Context)
	return nil
}

type kubernetesServiceManifest struct {
	Metadata struct {
		Name string `json:"name"`
		ID   string `json:"uid"`
	} `json:"metadata"`
}

// Inspect inspects a service
func (c *kubernetesDeploymentManifest) Inspect(service string) (*ServiceManifest, error) {
	kubectl = cluster.Kubectl().WithNamespace(c.Context, "default")
	out, err := kubectl.Run(c.Context, "get", "deployment/"+service, "-o", "json")
	if err != nil {
		return &ServiceManifest{}, err
	}
	var inspect kubernetesServiceManifest
	if err = json.Unmarshal([]byte(out), &inspect); err != nil {
		return &ServiceManifest{}, errors.Wrap(err, "Could not convert metadata to JSON")
	}
	return &ServiceManifest{
		ID:         inspect.Metadata.ID,
		Name:       strings.TrimPrefix(inspect.Metadata.Name, "/"),
		Connection: service,
		Hostname:   service,
	}, nil
}

// Remove remove services from deployment
func (c *kubernetesDeploymentManifest) Remove(services []string, env map[string]string) error {
	kubectl = cluster.Kubectl().WithNamespace(c.Context, "default")

	for _, service := range services {
		_, err := kubectl.Run(c.Context, "delete", "-k", fmt.Sprintf("../../../cli/config/kubernetes/overlays/%s", service))
		if err != nil {
			return err
		}
	}
	return nil
}
