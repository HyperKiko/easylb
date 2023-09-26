package util

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func WaitForDeploymentComplete(client client.Client, d *appsv1.Deployment) error {
	var reason string

	err := wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, 5*time.Minute, false, func(ctx context.Context) (bool, error) {
		err := client.Get(ctx, types.NamespacedName{Namespace: d.Namespace, Name: d.Name}, d)
		if err != nil {
			return false, err
		}
		if DeploymentComplete(d) {
			return true, nil
		}

		reason = fmt.Sprintf("deployment status: %#v", d.Status)

		return false, nil
	})

	if err == context.DeadlineExceeded {
		return fmt.Errorf("error waiting timeout: %s", reason)
	}
	if err != nil {
		return fmt.Errorf("error waiting for deployment %q status to match expectation: %v", d.Name, err)
	}

	return nil
}

func DeploymentComplete(deployment *appsv1.Deployment) bool {
	return deployment.Status.UpdatedReplicas == *(deployment.Spec.Replicas) &&
		deployment.Status.Replicas == *(deployment.Spec.Replicas) &&
		deployment.Status.AvailableReplicas == *(deployment.Spec.Replicas) &&
		deployment.Status.ObservedGeneration >= deployment.Generation
}

func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
