package clusterconfig

import (
	"net/http"
)

const AwsMetadataHost = "http://169.254.169.254/latest/meta-data/"

type Hyperscaler interface {
	IsAws() bool
}

type HyperscalerClient struct {
	http            *http.Client
	awsMetadataHost string
}

func NewHyperscalerClient(client *http.Client, awsMetadataHost string) *HyperscalerClient {
	return &HyperscalerClient{
		client,
		awsMetadataHost,
	}
}

func (hs *HyperscalerClient) IsAws() bool {
	r, err := hs.http.Get(hs.awsMetadataHost)
	if err != nil {
		return false
	}
	if r.StatusCode == http.StatusOK {
		return true
	}
	return false
}
