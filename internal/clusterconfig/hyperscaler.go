package clusterconfig

import (
	"net/http"
)

const AwsMetadataHost = "http://169.254.169.254/latest/meta-data/"

type Hyperscaler interface {
	IsAws() bool
}

type HyperscalerClient struct {
	HttpClient      *http.Client
	AwsMetadataHost string
}

func NewHyperscalerClient(client *http.Client) *HyperscalerClient {
	return &HyperscalerClient{
		client,
		AwsMetadataHost,
	}
}

func (hs *HyperscalerClient) IsAws() bool {
	r, err := hs.HttpClient.Get(hs.AwsMetadataHost)
	if err != nil {
		return false
	}
	if r.StatusCode == http.StatusOK {
		return true
	}
	return false
}
