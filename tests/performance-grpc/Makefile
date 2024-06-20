OS_ARCH ?= $(shell uname | awk '{print tolower($0)}')
KYMA_DOMAIN ?= $(shell kubectl config view -o json | jq '.clusters[0].cluster.server' | sed -e "s/https:\/\/api.//" -e 's/"//g')

hello.proto:
	@echo "Downloading proto definition for grpcbin"
	curl -sSL https://github.com/kyma-project/networking-dev-tools/raw/main/grpcbin/pkg/hello/hello.proto -o hello.proto

deploy-helm:
	helm dependency update load-testing
	helm upgrade -n load-testing --wait --install load-testing load-testing --set IngressGatewayIP=$(shell kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}') --set Domain=${KYMA_DOMAIN} --create-namespace --wait

grpc-load-test:
	@echo "Running gRPC load test"
	KYMA_DOMAIN=${KYMA_DOMAIN} go test -v -timeout 35m -run TestPerformanceGRPC

export-results:
	@echo "Exporting results"
	kubectl cp $(shell kubectl get pods --selector=app=load-testing-sftp -n load-testing -o jsonpath='{.items[0].metadata.name}'):/tmp results -n load-testing