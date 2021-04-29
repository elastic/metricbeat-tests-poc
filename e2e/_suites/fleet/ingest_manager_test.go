// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"os"
	"time"

	"github.com/cucumber/godog"
	"github.com/cucumber/messages-go/v10"
	"github.com/elastic/e2e-testing/cli/config"
	"github.com/elastic/e2e-testing/internal/common"
	"github.com/elastic/e2e-testing/internal/deploy"
	"github.com/elastic/e2e-testing/internal/installer"
	"github.com/elastic/e2e-testing/internal/kibana"
	"github.com/elastic/e2e-testing/internal/shell"
	"github.com/elastic/e2e-testing/internal/utils"
	log "github.com/sirupsen/logrus"
)

var imts IngestManagerTestSuite

func setUpSuite() {
	config.Init()

	kibanaClient, err := kibana.NewClient()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	deployer := deploy.NewClient()

	developerMode := shell.GetEnvBool("DEVELOPER_MODE")
	if developerMode {
		log.Info("Running in Developer mode 💻: runtime dependencies between different test runs will be reused to speed up dev cycle")
	}

	// check if base version is an alias
	v, err := utils.GetElasticArtifactVersion(common.AgentVersionBase)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"version": common.AgentVersionBase,
		}).Fatal("Failed to get agent base version, aborting")
	}
	common.AgentVersionBase = v

	common.TimeoutFactor = shell.GetEnvInteger("TIMEOUT_FACTOR", common.TimeoutFactor)
	common.AgentVersion = shell.GetEnv("BEAT_VERSION", common.AgentVersionBase)

	// check if version is an alias
	v, err = utils.GetElasticArtifactVersion(common.AgentVersion)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"version": common.AgentVersion,
		}).Fatal("Failed to get agent version, aborting")
	}
	common.AgentVersion = v

	common.StackVersion = shell.GetEnv("STACK_VERSION", common.StackVersion)
	v, err = utils.GetElasticArtifactVersion(common.StackVersion)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"version": common.StackVersion,
		}).Fatal("Failed to get stack version, aborting")
	}
	common.StackVersion = v

	common.KibanaVersion = shell.GetEnv("KIBANA_VERSION", "")
	if common.KibanaVersion == "" {
		// we want to deploy a released version for Kibana
		// if not set, let's use stackVersion
		common.KibanaVersion, err = utils.GetElasticArtifactVersion(common.StackVersion)
		if err != nil {
			log.WithFields(log.Fields{
				"error":   err,
				"version": common.KibanaVersion,
			}).Fatal("Failed to get kibana version, aborting")
		}
	}

	imts = IngestManagerTestSuite{
		Fleet: &FleetTestSuite{
			kibanaClient: kibanaClient,
			deployer:     deployer,
			Installers:   map[string]installer.ElasticAgentInstaller{}, // do not pre-initialise the map
			ctx:          context.Background(),
		},
		StandAlone: &StandAloneTestSuite{
			kibanaClient: kibanaClient,
		},
	}
}

func InitializeIngestManagerTestScenario(ctx *godog.ScenarioContext) {
	ctx.BeforeScenario(func(*messages.Pickle) {
		log.Trace("Before Fleet scenario")

		imts.StandAlone.Cleanup = false

		imts.Fleet.beforeScenario()
	})

	ctx.AfterScenario(func(*messages.Pickle, error) {
		log.Trace("After Fleet scenario")

		if imts.StandAlone.Cleanup {
			imts.StandAlone.afterScenario()
		}

		if imts.Fleet.Cleanup {
			imts.Fleet.afterScenario()
		}
	})

	ctx.Step(`^the "([^"]*)" process is in the "([^"]*)" state on the host$`, imts.processStateOnTheHost)

	imts.Fleet.contributeSteps(ctx)
	imts.StandAlone.contributeSteps(ctx)
}

func InitializeIngestManagerTestSuite(ctx *godog.TestSuiteContext) {
	deployer := deploy.NewClient()
	developerMode := shell.GetEnvBool("DEVELOPER_MODE")

	ctx.BeforeSuite(func() {
		setUpSuite()

		log.Trace("Bootstrapping Fleet Server")

		deployer.Bootstrap()

		serviceManifest, err := deployer.Inspect("fleet-server")
		if err != nil {
			log.WithField("manifest", serviceManifest).Fatal("Unable to grab service manifest")
		}
		imts.Fleet.FleetServerHostname = serviceManifest.Hostname

		log.WithField("manifest", serviceManifest).Trace("Discovered Fleet Server hostname")

		imts.Fleet.Version = common.AgentVersionBase
		imts.StandAlone.RuntimeDependenciesStartDate = time.Now().UTC()
	})

	ctx.AfterSuite(func() {
		if !developerMode {
			log.Debug("Destroying Fleet runtime dependencies")
			deployer.Destroy()
		}

		installers := imts.Fleet.Installers
		for k, v := range installers {
			agentPath := v.BinaryPath
			if _, err := os.Stat(agentPath); err == nil {
				err = os.Remove(agentPath)
				if err != nil {
					log.WithFields(log.Fields{
						"err":       err,
						"installer": k,
						"path":      agentPath,
					}).Warn("Elastic Agent binary could not be removed.")
				} else {
					log.WithFields(log.Fields{
						"installer": k,
						"path":      agentPath,
					}).Debug("Elastic Agent binary was removed.")
				}
			}
		}
	})
}
