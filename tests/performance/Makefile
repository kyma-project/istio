OS_ARCH ?= $(shell uname | awk '{print tolower($0)}')
KYMA_DOMAIN ?= $(shell kubectl config view -o json | jq '.clusters[0].cluster.server' | sed -e "s/https:\/\/api.//" -e 's/"//g')

load-testing/charts/plutono:
	mkdir load-testing/charts
	curl -sL https://github.com/credativ/helmcharts/releases/download/plutono-0.1.0/plutono-0.1.0.tgz > plutono.tgz
	tar -xf plutono.tgz -C load-testing/charts

deploy-helm:
	#kubectl wait --kubeconfig "${GARDENER_KUBECONFIG}" --for=jsonpath='{.status.lastOperation.state}'=Succeeded --timeout=600s "shoots/${CLUSTER_NAME}"
	helm dependency update load-testing
	helm upgrade --install load-testing load-testing --set IngressGatewayIP=$(shell kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}') --set Domain=${KYMA_DOMAIN} --create-namespace --wait

test-deploy:
	kubectl rollout status deployment/load-testing
	kubectl cp scripts/common.js $(shell kubectl get pods --selector=app=k6 -o jsonpath='{.items[0].metadata.name}'):run.js -c k6-alpine

test-performance: deploy-max-replicas-istio deploy-helm test-deploy
	kubectl patch deployment/load-testing --type merge --patch-file istio-disabled.yaml
	kubectl rollout status deployment/load-testing
	kubectl exec -q $(shell kubectl get pods --selector=app=k6 -o jsonpath='{.items[0].metadata.name}') -- k6 run run.js -d 1m --vus 500 --out influxdb=http://load-testing-influxdb:8086/k6 --log-output none --system-tags=method,name,status,tag -e DOMAIN=${KYMA_DOMAIN};\
	EXIT_CODE=$$?;\
	echo "Performance test without sidecar test completed with exit code $$EXIT_CODE"
	sleep 5
	kubectl cp $(shell kubectl get pods --selector=app=k6 -o jsonpath='{.items[0].metadata.name}'):summary.html summary-no-sidecar.html

	kubectl patch deployment/load-testing --type merge --patch-file istio-enabled.yaml
	kubectl rollout status deployment/load-testing
	kubectl exec -q $(shell kubectl get pods --selector=app=k6 -o jsonpath='{.items[0].metadata.name}') -- k6 run run.js -d 1m --vus 500 --out influxdb=http://load-testing-influxdb:8086/k6 --log-output none --system-tags=method,name,status,tag -e DOMAIN=${KYMA_DOMAIN};\
	EXIT_CODE=$$?;\
	echo "Performance test with sidecar test completed with exit code $$EXIT_CODE"
	sleep 5
	kubectl cp $(shell kubectl get pods --selector=app=k6 -o jsonpath='{.items[0].metadata.name}'):summary.html summary-sidecar.html

	cd hack/puppeteer && KYMA_DOMAIN=${KYMA_DOMAIN} ./export.sh

test-performance-web: deploy-max-replicas-istio deploy-helm test-deploy
	kubectl patch deployment/load-testing --type merge --patch-file istio-disabled.yaml
	kubectl rollout status deployment/load-testing
	kubectl exec -q $(shell kubectl get pods --selector=app=k6 -o jsonpath='{.items[0].metadata.name}') -- K6_WEB_DASHBOARD=true K6_WEB_DASHBOARD_EXPORT=summary.html k6 run run.js -d 1m --vus 500 --log-output none --system-tags=method,name,status,tag -e DOMAIN=${KYMA_DOMAIN};\
	EXIT_CODE=$$?;\
	echo "Performance test without sidecar test completed with exit code $$EXIT_CODE"
	sleep 5
	kubectl cp $(shell kubectl get pods --selector=app=k6 -o jsonpath='{.items[0].metadata.name}'):summary.html summary-no-sidecar.html

	kubectl patch deployment/load-testing --type merge --patch-file istio-enabled.yaml
	kubectl rollout status deployment/load-testing
	kubectl exec -q $(shell kubectl get pods --selector=app=k6 -o jsonpath='{.items[0].metadata.name}') -- K6_WEB_DASHBOARD=true K6_WEB_DASHBOARD_EXPORT=summary.html k6 run run.js -d 1m --vus 500 --log-output none --system-tags=method,name,status,tag -e DOMAIN=${KYMA_DOMAIN};\
	EXIT_CODE=$$?;\
	echo "Performance test with sidecar test completed with exit code $$EXIT_CODE"
	sleep 5
	kubectl cp $(shell kubectl get pods --selector=app=k6 -o jsonpath='{.items[0].metadata.name}'):summary.html summary-sidecar.html

deploy-max-replicas-istio:
	kubectl apply -f istio-max-replicas.yaml
	kubectl wait --for='jsonpath={.status.state}=Ready' --timeout=5m -n kyma-system istio default
	kubectl wait -n istio-system --for=jsonpath='{.status.loadBalancer.ingress}' --timeout=5m service/istio-ingressgateway

	domain=$(shell kubectl config view -o json | jq '.clusters[0].cluster.server' | sed -e "s/https:\/\/api.//" -e 's/"//g');\
	kubectl annotate service -n istio-system istio-ingressgateway "dns.gardener.cloud/dnsnames=*.${domain}" --overwrite

.PHONY: test-performance deploy-max-replicas-istio deploy-helm test-deploy
