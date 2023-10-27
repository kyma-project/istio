package clusterconfig

import (
	"net/http"
	"time"
)

const awsMetadataHost = "http://169.254.169.254"

// TODO: get rid of Interface in the name bellow

type AwsClientInterface interface {
	IsAws() bool
}
type AwsClient struct {
	*http.Client
}

func NewAwsClient() *AwsClient {
	return &AwsClient{
		&http.Client{Timeout: 5 * time.Second},
	}
}

func (ac *AwsClient) IsAws() bool {
	_, err := ac.Get(awsMetadataHost)
	if err != nil {
		return false
	}
	return true
}
