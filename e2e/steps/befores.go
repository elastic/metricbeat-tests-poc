package steps

import (
	"context"
	"strings"

	"github.com/elastic/e2e-testing/cli/services"
	"github.com/elastic/e2e-testing/cli/shell"
	log "github.com/sirupsen/logrus"
)

// AddAPMServicesForInstrumentation adds a Kibana and APM Server instances to the running project
func AddAPMServicesForInstrumentation(ctx context.Context, profile string, stackVersion string, needsKibana bool, env map[string]string) {
	serviceManager := services.NewServiceManager()

	apmServerURL := shell.GetEnv("APM_SERVER_URL", "")
	if strings.HasPrefix(apmServerURL, "http://localhost") {
		apmServices := []string{
			"apm-server",
		}

		if needsKibana {
			env["kibanaTag"] = stackVersion
			apmServices = append(apmServices, "kibana")
		}

		log.WithFields(log.Fields{
			"services": apmServices,
			"version":  stackVersion,
		}).Info("Starting APM services for instrumentation")

		env["apmServerTag"] = stackVersion
		err := serviceManager.AddServicesToCompose(ctx, profile, apmServices, env)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"env":   env,
			}).Warn("The APM Server and Kibana could not be started, but they are not needed by the tests. Continuing")
		}
	}
}
