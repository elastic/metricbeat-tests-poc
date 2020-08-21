package main

import (
	"fmt"
	"strings"

	"github.com/Jeffail/gabs/v2"
	curl "github.com/elastic/e2e-testing/cli/shell"
	log "github.com/sirupsen/logrus"
)

const ingestManagerIntegrationURL = kibanaBaseURL + "/api/ingest_manager/epm/packages/%s-%s"
const ingestManagerIntegrationDeleteURL = kibanaBaseURL + "/api/ingest_manager/package_policies/delete"
const ingestManagerIntegrationsURL = kibanaBaseURL + "/api/ingest_manager/epm/packages?experimental=true&category="
const ingestManagerIntegrationPoliciesURL = kibanaBaseURL + "/api/ingest_manager/package_policies"

// IntegrationPackage used to share information about a integration
type IntegrationPackage struct {
	packageConfigID string `json:"packageConfigId"`
	name            string `json:"name"`
	title           string `json:"title"`
	version         string `json:"version"`
}

// addIntegrationToPolicy sends a POST request to Ingest Manager adding an integration to a configuration
func addIntegrationToPolicy(integrationPackage IntegrationPackage, policyID string) (string, error) {
	postReq := createDefaultHTTPRequest(ingestManagerIntegrationPoliciesURL)

	data := `{
		"name":"` + integrationPackage.name + `-test-name",
		"description":"` + integrationPackage.title + `-test-description",
		"namespace":"default",
		"policy_id":"` + policyID + `",
		"enabled":true,
		"output_id":"",
		"inputs":[],
		"package":{
			"name":"` + integrationPackage.name + `",
			"title":"` + integrationPackage.title + `",
			"version":"` + integrationPackage.version + `"
		}
	}`
	postReq.Payload = []byte(data)
	body, err := curl.Post(postReq)
	if err != nil {
		log.WithFields(log.Fields{
			"body":    body,
			"error":   err,
			"url":     ingestManagerIntegrationPoliciesURL,
			"payload": data,
		}).Error("Could not add integration to configuration")
		return "", err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse response into JSON")
		return "", err
	}

	integrationConfigurationID := jsonParsed.Path("item.id").Data().(string)

	log.WithFields(log.Fields{
		"policyID":                   policyID,
		"integrationConfigurationID": integrationConfigurationID,
		"integration":                integrationPackage.name,
		"version":                    integrationPackage.version,
	}).Info("Integration added to the configuration")

	return integrationConfigurationID, nil
}

// deleteIntegrationFromPolicy sends a POST request to Ingest Manager deleting an integration from a configuration
func deleteIntegrationFromPolicy(integrationPackage IntegrationPackage, policyID string) error {
	postReq := createDefaultHTTPRequest(ingestManagerIntegrationDeleteURL)

	data := `{"packagePolicyIds":["` + integrationPackage.packageConfigID + `"]}`
	postReq.Payload = []byte(data)
	body, err := curl.Post(postReq)
	if err != nil {
		log.WithFields(log.Fields{
			"body":    body,
			"error":   err,
			"url":     ingestManagerIntegrationDeleteURL,
			"payload": data,
		}).Error("Could not delete integration from configuration")
		return err
	}

	log.WithFields(log.Fields{
		"policyID":        policyID,
		"integration":     integrationPackage.name,
		"packageConfigId": integrationPackage.packageConfigID,
		"version":         integrationPackage.version,
	}).Info("Integration deleted from the configuration")

	return nil
}

// getIntegrationLatestVersion sends a GET request to Ingest Manager for the existing integrations
// checking if the desired integration exists in the package registry. If so, it will
// return name and version (latest) of the integration
func getIntegrationLatestVersion(integrationName string) (string, string, error) {
	r := createDefaultHTTPRequest(ingestManagerIntegrationsURL)
	body, err := curl.Get(r)
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"error": err,
			"url":   ingestManagerIntegrationsURL,
		}).Error("Could not get Integrations")
		return "", "", err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse response into JSON")
		return "", "", err
	}

	// data streams should contain array of elements
	integrations := jsonParsed.Path("response").Children()

	log.WithFields(log.Fields{
		"count": len(integrations),
	}).Debug("Integrations retrieved")

	for _, integration := range integrations {
		name := integration.Path("name").Data().(string)
		if name == strings.ToLower(integrationName) {
			version := integration.Path("version").Data().(string)
			return name, version, nil
		}
	}

	return "", "", fmt.Errorf("The %s integration was not found", integrationName)
}

// installIntegration sends a POST request to Ingest Manager installing the assets for an integration
func installIntegrationAssets(integration string, version string) (IntegrationPackage, error) {
	url := fmt.Sprintf(ingestManagerIntegrationURL, integration, version)
	postReq := createDefaultHTTPRequest(url)

	body, err := curl.Post(postReq)
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"error": err,
			"url":   url,
		}).Error("Could not install assets for the integration")
		return IntegrationPackage{}, err
	}

	log.WithFields(log.Fields{
		"integration": integration,
		"version":     version,
	}).Debug("Assets for the integration where installed")

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse install response into JSON")
		return IntegrationPackage{}, err
	}
	response := jsonParsed.Path("response").Index(0)

	packageConfigID := response.Path("id").Data().(string)

	// get the integration again in the case it's already installed
	body, err = curl.Get(postReq)
	if err != nil {
		log.WithFields(log.Fields{
			"body":  body,
			"error": err,
			"url":   url,
		}).Error("Could not get the integration")
		return IntegrationPackage{}, err
	}

	jsonParsed, err = gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse get response into JSON")
		return IntegrationPackage{}, err
	}

	response = jsonParsed.Path("response")
	integrationPackage := IntegrationPackage{
		packageConfigID: packageConfigID,
		name:            response.Path("name").Data().(string),
		title:           response.Path("title").Data().(string),
		version:         response.Path("latestVersion").Data().(string),
	}

	return integrationPackage, nil
}
