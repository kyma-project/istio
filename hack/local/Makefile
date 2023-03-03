.PHONY: run-k3d-with-istio
run-k3d-with-istio:
	@make -C ./istio run

.PHONY: busola-enable
busola-enable:
	@make -C ./busola busola-enable

.PHONY: verify
verify:
	@make -C ./istio verify

.PHONY: run
run: run-k3d-with-istio busola-enable

.PHONY: test
test: run verify
	@echo '### istio operator operational'

.PHONY: stop
stop:
	@make -C ./istio stop
