OS_ARCH ?= $(shell uname | awk '{print tolower($0)}')
KYMA_DOMAIN ?= $(k config view -o json | jq '.clusters[0].cluster.server' | sed -e "s/https:\/\/api.//" -e 's/"//g')

bin/kyma:
	mkdir bin
	curl -Lo bin/kyma https://storage.googleapis.com/kyma-cli-unstable/kyma-${OS_ARCH}
	chmod +x bin/kyma

deploy: bin/kyma
	bin/kyma deploy -s main -p production --component cluster-essentials --component istio --component istio-resources --component certificates --ci
	touch deploy

deploy-helm:
	helm upgrade --install load-testing load-testing --set Domain=${KYMA_DOMAIN}
	touch deploy-helm

test-deploy:
	kubectl rollout status deployment/load-testing
	kubectl cp scripts/common.js $(shell kubectl get pods --selector=app=k6 -o jsonpath='{.items[0].metadata.name}'):run.js -c k6-alpine
	touch test-deploy

test-performance: deploy deploy-helm test-deploy
	kubectl scale --replicas=3 deployment/istio-ingressgateway -n istio-system
	kubectl patch deployment/load-testing --type merge --patch-file istio-disabled.yaml
	kubectl rollout status deployment/load-testing
	kubectl exec -it $(shell kubectl get pods --selector=app=k6 -o jsonpath='{.items[0].metadata.name}') -- k6 run run.js -d 1m --vus 500 --out influxdb=http://load-testing-influxdb:8086/k6 --system-tags=method,name,status,tag -e DOMAIN=${KYMA_DOMAIN}
	kubectl cp $(shell kubectl get pods --selector=app=k6 -o jsonpath='{.items[0].metadata.name}'):summary.html summary-no-sidecar.html

	kubectl scale --replicas=3 deployment/istio-ingressgateway -n istio-system
	kubectl patch deployment/load-testing --type merge --patch-file istio-enabled.yaml
	kubectl rollout status deployment/load-testing
	kubectl exec -it $(shell kubectl get pods --selector=app=k6 -o jsonpath='{.items[0].metadata.name}') -- k6 run run.js -d 1m --vus 500 --out influxdb=http://load-testing-influxdb:8086/k6 --system-tags=method,name,status,tag -e DOMAIN=${KYMA_DOMAIN}
	kubectl cp $(shell kubectl get pods --selector=app=k6 -o jsonpath='{.items[0].metadata.name}'):summary.html summary-sidecar.html

.PHONY: test-performance
