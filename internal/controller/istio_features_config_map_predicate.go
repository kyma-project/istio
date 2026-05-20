package controller

import (
	"context"

	"github.com/kyma-project/istio/operator/internal/istiofeatures"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

// IstioFeaturesConfigMapEventHandler is a controller-runtime EventHandler that triggers reconciliation
// of the Istio CR whenever the istio-features ConfigMap is created, updated, or deleted.
type IstioFeaturesConfigMapEventHandler struct{}

func (h IstioFeaturesConfigMapEventHandler) isIstioFeaturesConfigMap(obj client.Object) bool {
	return obj.GetName() == istiofeatures.ConfigMapName && obj.GetNamespace() == istiofeatures.ConfigMapNamespace
}

func (h IstioFeaturesConfigMapEventHandler) enqueue(w workqueue.TypedRateLimitingInterface[controllerruntime.Request]) {
	w.Add(controllerruntime.Request{NamespacedName: types.NamespacedName{Namespace: namespace, Name: "default"}})
}

func (h IstioFeaturesConfigMapEventHandler) Create(_ context.Context, ev event.TypedCreateEvent[client.Object], w workqueue.TypedRateLimitingInterface[controllerruntime.Request]) {
	if h.isIstioFeaturesConfigMap(ev.Object) {
		h.enqueue(w)
	}
}

func (h IstioFeaturesConfigMapEventHandler) Update(_ context.Context, ev event.TypedUpdateEvent[client.Object], w workqueue.TypedRateLimitingInterface[controllerruntime.Request]) {
	if h.isIstioFeaturesConfigMap(ev.ObjectNew) {
		h.enqueue(w)
	}
}

func (h IstioFeaturesConfigMapEventHandler) Delete(_ context.Context, ev event.TypedDeleteEvent[client.Object], w workqueue.TypedRateLimitingInterface[controllerruntime.Request]) {
	if h.isIstioFeaturesConfigMap(ev.Object) {
		h.enqueue(w)
	}
}

func (h IstioFeaturesConfigMapEventHandler) Generic(_ context.Context, _ event.TypedGenericEvent[client.Object], _ workqueue.TypedRateLimitingInterface[controllerruntime.Request]) {
}
