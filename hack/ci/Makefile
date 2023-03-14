PROJECT_ROOT ?= ../..
PROJECT_istio_LOCAL ?= ${PROJECT_ROOT}/hack/local/istio

include ${PROJECT_ROOT}/.env

ifndef MODULE_SHA
    MODULE_SHA = ${shell git rev-parse --short HEAD}
endif

.PHONY: kyma
kyma:
	@make -C ${PROJECT_ROOT} kyma

.PHONY: module-build
module-build: 
	@make -C ${PROJECT_ROOT} module-build \
		IMG=${IMG} \
		MODULE_REGISTRY=${MODULE_REGISTRY} \
		MODULE_VERSION=${MODULE_VERSION}-${MODULE_SHA} \

.PHONY: module-template-arifact
module-template-artifact:
	@cp ${PROJECT_ROOT}/template.yaml ${ARTIFACTS}

.PHONY: module-template-print
module-template-print:
	@echo "\n~~~~~~~~~~~~BEGINING OF MODULE TEMPLATE~~~~~~~~~~~~~~"
	@cat ${PROJECT_ROOT}/template.yaml
	@echo "\n~~~~~~~~~~~~~~~END OF MODULE TEMPLATE~~~~~~~~~~~~~~~~"

# rename it from 'all' to something more accurate like 'module-build-and-print'
.PHONY: all
all: module-build module-template-artifact module-template-print

.PHONY: integration-test
integration-test:
	@echo "integration tests not implemented yet"
