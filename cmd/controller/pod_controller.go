package main

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/hyperkiko/easylb/pkg/constant"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type reconcilePod struct {
	client client.Client
}

var _ reconcile.Reconciler = &reconcilePod{}

func (r *reconcilePod) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := log.FromContext(ctx)

	pod := &corev1.Pod{}
	err := r.client.Get(ctx, request.NamespacedName, pod)
	if errors.IsNotFound(err) {
		log.Error(nil, "Could not find Pod")
		return reconcile.Result{}, nil
	}

	if err != nil {
		return reconcile.Result{}, fmt.Errorf("could not fetch Pod: %+v", err)
	}

	if _, ok := pod.Annotations[constant.ManagedAnnotation]; !ok {
		return reconcile.Result{}, nil
	}

	if pod.Status.Phase != corev1.PodRunning {
		return reconcile.Result{}, nil
	}

	svc := &corev1.Service{}
	if err = r.client.Get(ctx, types.NamespacedName{
		Namespace: pod.Annotations[constant.LoadBalancerForNamespaceAnnotation],
		Name:      pod.Annotations[constant.LoadBalancerForNameAnnotation],
	}, svc); err != nil {
		return reconcile.Result{}, fmt.Errorf("could not find Service: %+v", err)
	}

	node := &corev1.Node{}
	if err = r.client.Get(ctx, types.NamespacedName{Name: pod.Spec.NodeName}, node); err != nil {
		return reconcile.Result{}, fmt.Errorf("could not get node that pod is running on: %+v", err)
	}

	ip := pod.Status.HostIP
	for _, nodeAddr := range node.Status.Addresses {
		ip = nodeAddr.Address
		if nodeAddr.Type == corev1.NodeExternalIP {
			break
		}
	}

	svc.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{{IP: ip}}

	if err = r.client.Status().Update(ctx, svc); err != nil {
		return reconcile.Result{}, fmt.Errorf("could not set loadbalancer ip on service: %+v", err)
	}

	return reconcile.Result{}, nil
}
