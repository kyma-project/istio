package proxy_config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
)

type ListenerConfig struct {
	Configs []DynamicListener `json:"configs"`
}

type DynamicListener struct {
	Type        string      `json:"@type"`
	Name        string      `json:"name"`
	ActiveState ActiveState `json:"active_state"`
}

type ActiveState struct {
	Listener Listener `json:"listener"`
}

type Listener struct {
	Type    string        `json:"@type"`
	Name    string        `json:"name"`
	Address Address       `json:"address"`
	Filters []FilterChain `json:"filter_chains"`
}

type Address struct {
	SocketAddress SocketAddress `json:"socket_address"`
}

type SocketAddress struct {
	Address   string `json:"address"`
	PortValue int    `json:"port_value"`
}

type FilterChain struct {
	Filters []Filter `json:"filters"`
}

type Filter struct {
	Name        string      `json:"name"`
	TypedConfig TypedConfig `json:"typed_config"`
}

type TypedConfig struct {
	Type       string `json:"@type"`
	StatPrefix string `json:"stat_prefix,omitempty"`
	Cluster    string `json:"cluster,omitempty"`
}

type ListenerData struct {
	ListenerName  string
	SocketAddress string
	SocketPort    int
	StatPrefix    string
	Cluster       string
}

// extractListenerData parses a JSON config dump and extracts listener information.
// It unmarshals the config, iterates through listeners and their filter chains,
// and returns a slice of ListenerData containing socket addresses and cluster details.
func extractListenerData(configFile []byte) ([]ListenerData, error) {
	var config ListenerConfig
	if err := json.Unmarshal(configFile, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal listener config: %w", err)
	}

	var result []ListenerData

	for _, listener := range config.Configs {
		addr := listener.ActiveState.Listener.Address.SocketAddress.Address
		port := listener.ActiveState.Listener.Address.SocketAddress.PortValue

		for _, chain := range listener.ActiveState.Listener.Filters {
			for _, filter := range chain.Filters {
				if filter.TypedConfig.Cluster != "" || filter.TypedConfig.StatPrefix != "" {
					result = append(result, ListenerData{
						ListenerName:  listener.Name,
						SocketAddress: addr,
						SocketPort:    port,
						StatPrefix:    filter.TypedConfig.StatPrefix,
						Cluster:       filter.TypedConfig.Cluster,
					})
				}
			}
		}
	}

	return result, nil
}

// getProxyConfigDump executes a pilot-agent request to retrieve the proxy configuration dump
// for a specific resource from a pod. It returns the config dump as bytes or an error if the
// request fails.
func getProxyConfigDump(t *testing.T, podName, podNamespace, resource string) ([]byte, error) {
	t.Helper()

	r, err := client.ResourcesClient(t)
	if err != nil {
		return nil, fmt.Errorf("could not create resources client: %w", err)
	}

	cmd := []string{"pilot-agent", "request", "GET", fmt.Sprintf("config_dump?format=json&resource=%s", resource)}

	var stdout, stderr bytes.Buffer
	err = r.ExecInPod(t.Context(), podNamespace, podName, "istio-proxy", cmd, &stdout, &stderr)
	if err != nil {
		t.Logf("[%s] stderr: %v", podName, strings.TrimSpace(stderr.String()))
		return nil, fmt.Errorf("exec failed: %w", err)
	}

	// Save config dump as artifact
	saveProxyConfigDump(t, podName, stdout.Bytes())

	return stdout.Bytes(), nil
}

// GetDynamicListeners retrieves all dynamic listeners from the proxy config dump of a pod.
// Returns a list of ListenerData containing socket addresses and cluster information.
func GetDynamicListeners(t *testing.T, podName, podNamespace string) ([]ListenerData, error) {
	t.Helper()

	t.Logf("Getting proxy-config from pod: %s/%s", podNamespace, podName)
	data, err := getProxyConfigDump(t, podName, podNamespace, "dynamic_listeners")
	if err != nil {
		return nil, fmt.Errorf("failed to get proxy config dump: %w", err)
	}

	t.Logf("Extracting dynamic_listeners data from proxy-config dump")
	listeners, err := extractListenerData(data)
	if err != nil {
		return nil, fmt.Errorf("failed to extract listener data: %w", err)
	}

	return listeners, nil
}

// FindListenerByHostAndPort finds a listener matching the given host and port.
// Returns the ListenerData if found, or nil if not found.
func FindListenerByHostAndPort(listeners []ListenerData, host string, port int) *ListenerData {
	for i, l := range listeners {
		if strings.Contains(l.Cluster, host) && l.SocketPort == port {
			return &listeners[i]
		}
	}
	return nil
}

// The tests uploads an artifact of proxy config dump for debugging purposes. It creates a directory structure based on the test name and timestamp, and saves the config dump as a JSON file named after the pod. If any step fails, it logs a warning but does not return an error to avoid disrupting the test flow.
const (
	baseArtifactDir = "test-artifacts"
	proxyConfigDir  = "proxy_config_dumps"
)

var (
	testRunTimestamp string
	timestampOnce    sync.Once
)

func getTestRunTimestamp() string {
	timestampOnce.Do(func() {
		testRunTimestamp = time.Now().Format("02_01_2006-15_04_05CET")
	})
	return testRunTimestamp
}

func sanitizePathComponent(name string) string {
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_",
		"|", "_", " ", "_", "(", "", ")", "", ",", "",
	)
	return replacer.Replace(name)
}

func saveProxyConfigDump(t *testing.T, podName string, data []byte) {
	t.Helper()

	testName := sanitizePathComponent(t.Name())
	dir := filepath.Join(".", baseArtifactDir, getTestRunTimestamp(), testName, proxyConfigDir)

	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Logf("Warning: failed to create artifact dir: %v", err)
		return
	}

	filename := fmt.Sprintf("%s-proxy_config_dump.json", sanitizePathComponent(podName))
	filePath := filepath.Join(dir, filename)

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Logf("Warning: failed to save proxy config dump: %v", err)
		return
	}

	t.Logf("Saved proxy config dump: %s", filePath)
}
