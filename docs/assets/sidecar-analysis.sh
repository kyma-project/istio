#!/bin/bash

set -eou pipefail

target_namespace="${1:-}"

log_pods_with () {
    namespace=$1
    label=${2:-}

    if [ -n "$label" ]; then
        cmd="kubectl get pod -l $label -n $namespace -o jsonpath='{.items[*].metadata.name}'"
    else
        cmd="kubectl get pod -n $namespace -o jsonpath='{.items[*].metadata.name}'"
    fi

    pods_out_of_istio=$(eval $cmd)
    for pod in $pods_out_of_istio
    do
        if [ "$target_namespace" == "" ]; then
            echo "    - $namespace/$pod"
        else
            echo "  - $pod"
        fi
    done
}

if [ -z "$target_namespace" ]; then
    echo "See all Pods that are not part of the Istio service mesh:"

    echo "  Pods in namespaces labeled with \"istio-injection=disabled\":"
    disabled_namespaces=$(kubectl get ns -l "istio-injection=disabled" -o jsonpath='{.items[*].metadata.name}')

    for ns in $disabled_namespaces
    do
        if [ "$ns" != "kube-system" ] && [ "$ns" != "kyma-system" ]; then
            log_pods_with $ns
        fi
    done

    echo "  Pods labeled with \"sidecar.istio.io/inject=false\" in namespaces labeled with \"istio-injection=enabled\":"
    enabled_ns=$(kubectl get ns -l "istio-injection=enabled" -o jsonpath='{.items[*].metadata.name}')
    for ns in $enabled_ns
    do
        if [ "$ns" != "kube-system" ] && [ "$ns" != "kyma-system" ]; then
            log_pods_with $ns "sidecar.istio.io/inject=false"
        fi
    done

    echo "  Pods labeled with \"sidecar.istio.io/inject=true\" in not labeled namespaces:"
    not_labeled_ns=$(kubectl get ns -l "istio-injection!=disabled, istio-injection!=enabled" -o jsonpath='{.items[*].metadata.name}')
    for ns in $not_labeled_ns
    do
        if [ "$ns" != "kube-system" ] && [ "$ns" != "kyma-system" ]; then
            log_pods_with $ns "sidecar.istio.io/inject!=true"
        fi
    done
else 
    ns_label=$(kubectl get ns $target_namespace -o jsonpath='{.metadata.labels.istio-injection}')
    echo "Pods out of istio mesh in namespace $target_namespace:"
    if [ "$ns_label" == "enabled" ]; then
        log_pods_with $target_namespace "sidecar.istio.io/inject=false"
    elif [ "$ns_label" == "disabled" ]; then
        log_pods_with $target_namespace
    else
        log_pods_with $target_namespace "sidecar.istio.io/inject!=true"
    fi
fi
