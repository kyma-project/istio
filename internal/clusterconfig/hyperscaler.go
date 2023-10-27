package clusterconfig

func IsHyperscalerAWS(awsClient AwsClientInterface) bool {
	return awsClient.IsAws()
}
