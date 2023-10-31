package clusterconfig

import (
	"net/http"
	"time"
)

const awsMetadataHost = "http://169.254.169.254/latest/meta-data/"

type HyperscalerClient interface {
	IsAws() bool
}

type Hyperscaler struct {
	http *http.Client
}

func NewHyperscalerClient() *Hyperscaler {
	return &Hyperscaler{
		&http.Client{Timeout: 1 * time.Second},
	}
}

func (hs *Hyperscaler) IsAws() bool {
	r, err := hs.http.Get(awsMetadataHost)
	if err != nil {
		return false
	}
	if r.StatusCode == http.StatusOK {
		return true
	}
	return false
}
