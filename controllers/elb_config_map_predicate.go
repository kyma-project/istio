package controllers

import (
	"context"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

//ElbConfigMapEventHandler is a controller-runtime EventHandler that returns true if the deleted corev1.ConfigMap is an ELB ConfigMap

type ElbConfigMapEventHandler struct{}

func (e ElbConfigMapEventHandler) Create(_ context.Context, _ event.TypedCreateEvent[client.Object], _ workqueue.TypedRateLimitingInterface[controllerruntime.Request]) {
}

func (e ElbConfigMapEventHandler) Update(_ context.Context, _ event.TypedUpdateEvent[client.Object], _ workqueue.TypedRateLimitingInterface[controllerruntime.Request]) {
}

func (e ElbConfigMapEventHandler) Delete(_ context.Context, ev event.TypedDeleteEvent[client.Object], w workqueue.TypedRateLimitingInterface[controllerruntime.Request]) {
	if ev.Object.GetNamespace() == "istio-system" && ev.Object.GetName() == "elb-deprecated" {
		w.Add(controllerruntime.Request{NamespacedName: types.NamespacedName{Namespace: namespace, Name: "default"}})
	}
	return
}

func (e ElbConfigMapEventHandler) Generic(_ context.Context, _ event.TypedGenericEvent[client.Object], _ workqueue.TypedRateLimitingInterface[controllerruntime.Request]) {
}
