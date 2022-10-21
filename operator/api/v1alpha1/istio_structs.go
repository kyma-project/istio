package v1alpha1

type ExtensionProvider struct {
	Path string `json:"path,omitempty"`
	Name string `json:"name,omitempty"`
}

type GatewayTopology struct {
	NumTrustedProxies        int    `json:"numTrustedProxies,omitempty"`
	ForwardClientCertDetails string `json:"forwardClientCertDetails,omitempty"`
}

type Zipkin struct {
	Address string `json:"address,omitempty"`
}

type Tracing struct {
	Sampling int    `json:"sampling,omitempty"`
	Zipkin   Zipkin `json:"zipkin,omitempty"`
}

type MeshConfig struct {
	GatewayTopology                 GatewayTopology     `json:"gatewayTopology,omitempty"`
	ExtensionProviders              []ExtensionProvider `json:"extensionProviders,omitempty"`
	AccessLogEncoding               string              `json:"accessLogEncoding,omitempty"`
	AccessLogFile                   string              `json:"accessLogFile,omitempty"`
	Tracing                         Tracing             `json:"tracing,omitempty"`
	HoldApplicationUntilProxyStarts bool                `json:"holdApplicationUntilProxyStarts,omitempty"`
}
