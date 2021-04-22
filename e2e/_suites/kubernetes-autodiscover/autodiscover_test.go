package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/cucumber/godog"
	messages "github.com/cucumber/messages-go/v10"
	log "github.com/sirupsen/logrus"

	"github.com/elastic/e2e-testing/internal/shell"
)

const defaultBeatVersion = "8.0.0-SNAPSHOT"
const defaultEventsWaitTimeout = 120 * time.Second
const defaultDeployWaitTimeout = 120 * time.Second

type podsManager struct {
	kubectl kubernetesControl
	ctx     context.Context
}

func (m *podsManager) executeTemplateFor(podName string, writer io.Writer, options []string) error {
	path := filepath.Join("testdata/templates", sanitizeName(podName)+".yml.tmpl")

	usedOptions := make(map[string]bool)
	funcs := template.FuncMap{
		"option": func(o string) bool {
			usedOptions[o] = true
			for _, option := range options {
				if o == option {
					return true
				}
			}
			return false
		},
		"beats_version": func() string {
			return shell.GetEnv("BEAT_VERSION", defaultBeatVersion)
		},
		"namespace": func() string {
			return m.kubectl.Namespace
		},
		// Can be used to add owner references so cluster-level resources
		// are removed when removing the namespace.
		"namespace_uid": func() string {
			return m.kubectl.NamespaceUID
		},
	}

	t, err := template.New(filepath.Base(path)).Funcs(funcs).ParseFiles(path)
	if os.IsNotExist(err) {
		log.Debugf("template %s does not exist", path)
		return godog.ErrPending
	}
	if err != nil {
		return fmt.Errorf("parsing template %s: %w", path, err)
	}

	err = t.ExecuteTemplate(writer, filepath.Base(path), nil)
	if err != nil {
		return fmt.Errorf("executing template %s: %w", path, err)
	}

	for _, option := range options {
		if _, used := usedOptions[option]; !used {
			log.Debugf("option '%s' is not used in template for '%s'", option, podName)
			return godog.ErrPending
		}
	}

	return nil
}

func (m *podsManager) isDeleted(podName string, options []string) error {
	var buf bytes.Buffer
	err := m.executeTemplateFor(podName, &buf, options)
	if err != nil {
		return err
	}

	_, err = m.kubectl.RunWithStdin(m.ctx, &buf, "delete", "-f", "-")
	if err != nil {
		return fmt.Errorf("failed to delete '%s': %w", podName, err)
	}
	return nil
}

func (m *podsManager) isDeployed(podName string, options []string) error {
	var buf bytes.Buffer
	err := m.executeTemplateFor(podName, &buf, options)
	if err != nil {
		return err
	}

	_, err = m.kubectl.RunWithStdin(m.ctx, &buf, "apply", "-f", "-")
	if err != nil {
		return fmt.Errorf("failed to deploy '%s': %w", podName, err)
	}
	return nil
}

func (m *podsManager) isRunning(podName string, options []string) error {
	err := m.isDeployed(podName, options)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(m.ctx, defaultDeployWaitTimeout)
	defer cancel()

	_, err = m.getPodInstances(ctx, podName)
	if err != nil {
		return fmt.Errorf("waiting for instance of '%s': %w", podName, err)
	}
	return nil
}

func (m *podsManager) resourceIs(podName string, state string, options ...string) error {
	switch state {
	case "running":
		return m.isRunning(podName, options)
	case "deployed":
		return m.isDeployed(podName, options)
	case "deleted":
		return m.isDeleted(podName, options)
	default:
		return godog.ErrPending
	}
}

func (m *podsManager) collectsEventsWith(podName string, condition string) error {
	_, _, ok := splitCondition(condition)
	if !ok {
		return fmt.Errorf("invalid condition '%s'", condition)
	}

	tmpDir, err := ioutil.TempDir(os.TempDir(), "test-")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	ctx, cancel := context.WithTimeout(m.ctx, defaultEventsWaitTimeout)
	defer cancel()

	instances, err := m.getPodInstances(ctx, podName)
	if err != nil {
		return fmt.Errorf("failed to get pod name: %w", err)
	}

	containerPath := fmt.Sprintf("%s/%s:/tmp/beats-events", m.kubectl.Namespace, instances[0])
	localPath := filepath.Join(tmpDir, "events")
	for {
		_, err := m.kubectl.Run(ctx, "cp", "--no-preserve", containerPath, localPath)
		if err == nil {
			ok, err := containsEventsWith(localPath, condition)
			if ok {
				break
			}
			if err != nil {
				log.Debugf("Error checking if %v contains %v: %v", localPath, condition, err)
			}
		} else {
			log.Debugf("Failed to copy events from %s to %s: %s", containerPath, localPath, err)
		}

		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for events with %s", condition)
		}
	}

	return nil
}

func (m *podsManager) getPodInstances(ctx context.Context, podName string) ([]string, error) {
	app := sanitizeName(podName)
	for {
		output, err := m.kubectl.Run(ctx, "get", "pods",
			"-l", "k8s-app="+app,
			"--template", `{{range .items}}{{ if eq .status.phase "Running" }}{{.metadata.name}}{{"\n"}}{{ end }}{{end}}`)
		if err != nil {
			return nil, err
		}
		if output != "" {
			instances := strings.Split(strings.TrimSpace(output), "\n")
			return instances, nil
		}

		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for running pods with label k8s-app=%s", app)
		}
	}
}

func splitCondition(c string) (key string, value string, ok bool) {
	fields := strings.SplitN(c, ":", 2)
	if len(fields) != 2 || len(fields[0]) == 0 {
		return
	}

	return fields[0], fields[1], true
}

func flattenMap(m map[string]interface{}) map[string]interface{} {
	flattened := make(map[string]interface{})
	for k, v := range m {
		switch child := v.(type) {
		case map[string]interface{}:
			childMap := flattenMap(child)
			for ck, cv := range childMap {
				flattened[k+"."+ck] = cv
			}
		default:
			flattened[k] = v
		}
	}
	return flattened
}

func containsEventsWith(path string, condition string) (bool, error) {
	key, value, ok := splitCondition(condition)
	if !ok {
		return false, fmt.Errorf("invalid condition '%s'", condition)
	}

	f, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	for decoder.More() {
		var event map[string]interface{}
		err := decoder.Decode(&event)
		if err == io.EOF {
			break
		}
		if err != nil {
			return false, fmt.Errorf("decoding event: %w", err)
		}

		event = flattenMap(event)
		if v, ok := event[key]; ok && fmt.Sprint(v) == value {
			return true, nil
		}
	}

	return false, nil
}

func sanitizeName(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), " ", "-")
}

func waitDuration(ctx context.Context, d string) error {
	duration, err := time.ParseDuration(d)
	if err != nil {
		return fmt.Errorf("invalid duration %s: %w", d, err)
	}

	select {
	case <-time.After(duration):
	case <-ctx.Done():
	}

	return nil
}

func (m *podsManager) stopsCollectingEvents(podName string) error {
	return godog.ErrPending
}

var cluster kubernetesCluster

func InitializeTestSuite(ctx *godog.TestSuiteContext) {
	suiteContext, cancel := context.WithCancel(context.Background())
	log.DeferExitHandler(cancel)

	ctx.BeforeSuite(func() {
		err := cluster.initialize(suiteContext)
		if err != nil {
			log.WithError(err).Fatal("Failed to initialize cluster")
		}
		log.DeferExitHandler(func() {
			cluster.cleanup(suiteContext)
		})
	})

	ctx.AfterSuite(func() {
		cluster.cleanup(suiteContext)
		cancel()
	})
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	scenarioCtx, cancel := context.WithCancel(context.Background())
	log.DeferExitHandler(cancel)

	var kubectl kubernetesControl
	var pods podsManager
	ctx.BeforeScenario(func(*messages.Pickle) {
		kubectl = cluster.Kubectl().WithNamespace(scenarioCtx, "")
		if kubectl.Namespace != "" {
			log.Debugf("Running scenario in namespace: %s", kubectl.Namespace)
		}
		pods.kubectl = kubectl
		pods.ctx = scenarioCtx
		log.DeferExitHandler(func() { kubectl.Cleanup(scenarioCtx) })
	})
	ctx.AfterScenario(func(*messages.Pickle, error) {
		kubectl.Cleanup(scenarioCtx)
		cancel()
	})

	ctx.Step(`^"([^"]*)" have passed$`, func(d string) error { return waitDuration(scenarioCtx, d) })

	ctx.Step(`^"([^"]*)" is ([a-z]*)$`, func(name, state string) error {
		return pods.resourceIs(name, state)
	})
	ctx.Step(`^"([^"]*)" is ([a-z]*) with "([^"]*)"$`, func(name, state, option string) error {
		return pods.resourceIs(name, state, option)
	})

	ctx.Step(`^"([^"]*)" collects events with "([^"]*:[^"]*)"$`, pods.collectsEventsWith)
	ctx.Step(`^"([^"]*)" stops collecting events$`, pods.stopsCollectingEvents)
}