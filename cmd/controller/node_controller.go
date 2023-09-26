package main

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/hyperkiko/easylb/pkg/constant"
)

type reconcileNode struct {
	client client.Client
}

var _ reconcile.Reconciler = &reconcileNode{}

func (r *reconcileNode) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := log.FromContext(ctx)

	node := &corev1.Node{}
	err := r.client.Get(ctx, request.NamespacedName, node)
	if errors.IsNotFound(err) {
		log.Error(nil, "Could not find Node")
		return reconcile.Result{}, nil
	}

	if err != nil {
		return reconcile.Result{}, fmt.Errorf("could not fetch Node: %+v", err)
	}

	for _, nodeAddr := range node.Status.Addresses {
		if nodeAddr.Type == corev1.NodeExternalIP {
			if _, ok := node.Labels[constant.ExternalIPLabel]; !ok {
				node.Labels[constant.ExternalIPLabel] = ""
				if err = r.client.Update(ctx, node); err != nil {
					return reconcile.Result{}, fmt.Errorf("could not label node: %+v", err)
				}
			}
		}
	}

	return reconcile.Result{}, nil
}
