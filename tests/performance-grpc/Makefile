OS_ARCH ?= $(shell uname | awk '{print tolower($0)}')
KYMA_DOMAIN ?= $(shell kubectl config view -o json | jq '.clusters[0].cluster.server' | sed -e "s/https:\/\/api.//" -e 's/"//g')

deploy-helm:
	helm dependency update load-testing
	helm upgrade -n load-testing --wait --install load-testing load-testing --set IngressGatewayIP=$(shell kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}') --set Domain=${KYMA_DOMAIN} --create-namespace --wait

grpc-load-test:
	@echo "Running gRPC load test"
	@go run main.go -address ${KYMA_DOMAIN}
