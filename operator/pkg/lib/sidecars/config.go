package sidecars

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/client-go/kubernetes"
)

// IstioProxyConfig stores input information for IstioProxyReset.
type IstioProxyConfig struct {
	// Reconcile action context
	Context context.Context

	// ImagePrefix of Istio
	ImagePrefix string

	// ImageVersion of Istio
	ImageVersion string

	// Kubeclient for k8s cluster operations
	Kubeclient kubernetes.Interface

	// Debug mode
	Debug bool

	// Logger to be used
	Log logr.Logger

	// Is Sidecar Injection enabled by default
	SidecarInjectionByDefaultEnabled bool

	// Is CNI enabled on the cluster
	CNIEnabled bool
}
