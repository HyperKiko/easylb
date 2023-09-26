package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/hyperkiko/easylb/pkg/constant"
	"github.com/hyperkiko/easylb/pkg/util"
)

func generateLoadBalancerName(svc *corev1.Service) string {
	return "easylb-lb-" + svc.Namespace + "-" + svc.Name
}

func generatePorts(ports []corev1.ServicePort) []corev1.ContainerPort {
	res := make([]corev1.ContainerPort, len(ports))
	for i, port := range ports {
		res[i].ContainerPort = port.Port
		res[i].HostPort = port.Port
		res[i].Name = port.Name
		res[i].Protocol = port.Protocol
	}
	return res
}

func generateArgument(ports []corev1.ServicePort, clusterIP string) string {
	var value string
	for _, port := range ports {
		value += clusterIP + " " + // IP
			strconv.Itoa(int(port.Port)) + " " + // Port
			strings.ToLower(string(port.Protocol)) + " " // Protocol
	}
	return value
}

type reconcileService struct {
	client client.Client
}

var _ reconcile.Reconciler = &reconcileService{}

func (r *reconcileService) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := log.FromContext(ctx)

	svc := &corev1.Service{}
	err := r.client.Get(ctx, request.NamespacedName, svc)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Error(nil, "Could not find Service")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("could not fetch Service: %+v", err)
	}

	if !svc.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.ReconcileDelete(ctx, svc)
	}

	if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		deployment := &appsv1.Deployment{}

		if err := r.client.Get(ctx, types.NamespacedName{Namespace: constant.Namespace, Name: generateLoadBalancerName(svc)}, deployment); err != nil {
			if errors.IsNotFound(err) {
				return reconcile.Result{}, nil
			}
			return reconcile.Result{}, fmt.Errorf("could not get deployment: %+v", err)
		}
		if err := r.client.Delete(ctx, deployment); err != nil {
			return reconcile.Result{}, fmt.Errorf("could not delete deployment: %+v", err)
		}

		return reconcile.Result{}, nil
	}

	if !util.ContainsString(svc.GetFinalizers(), constant.FinalizerName) {
		controllerutil.AddFinalizer(svc, constant.FinalizerName)
		if err := r.client.Update(ctx, svc); err != nil {
			return reconcile.Result{}, fmt.Errorf("could not register finalizer: %+v", err)
		}
	}

	deploymentNamespacedName := types.NamespacedName{Namespace: constant.Namespace, Name: generateLoadBalancerName(svc)}
	deployment := &appsv1.Deployment{}
	alreadyExists := true
	if err = r.client.Get(ctx, deploymentNamespacedName, deployment); errors.IsNotFound(err) {
		alreadyExists = false
	} else if err != nil {
		return reconcile.Result{}, fmt.Errorf("could not fetch Deployment: %+v", err)
	}
	privileged := true
	deployment = &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generateLoadBalancerName(svc),
			Namespace: constant.Namespace,
			Annotations: map[string]string{
				constant.ManagedAnnotation:                  "",
				constant.LoadBalancerForNamespaceAnnotation: svc.Namespace,
				constant.LoadBalancerForNameAnnotation:      svc.Name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": generateLoadBalancerName(svc),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"name": generateLoadBalancerName(svc)},
					Annotations: map[string]string{
						constant.ManagedAnnotation:                  "",
						constant.LoadBalancerForNamespaceAnnotation: svc.Namespace,
						constant.LoadBalancerForNameAnnotation:      svc.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  generateLoadBalancerName(svc),
						Image: constant.LoadBalancerImage,
						Ports: generatePorts(svc.Spec.Ports),
						Args:  []string{generateArgument(svc.Spec.Ports, svc.Spec.ClusterIP)},
						SecurityContext: &corev1.SecurityContext{
							Capabilities: &corev1.Capabilities{
								Add: []corev1.Capability{"NET_ADMIN"},
							},
						},
					}},
					InitContainers: []corev1.Container{{
						Name:  "init-" + generateLoadBalancerName(svc),
						Image: constant.EnableIPForwardImage,
						SecurityContext: &corev1.SecurityContext{
							Privileged: &privileged,
						},
					}},
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{{
									MatchExpressions: []corev1.NodeSelectorRequirement{{
										Key:      constant.ExcludeNodeLabel,
										Operator: corev1.NodeSelectorOpDoesNotExist,
									}},
								}},
							},
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
								{
									Weight: 2,
									Preference: corev1.NodeSelectorTerm{
										MatchExpressions: []corev1.NodeSelectorRequirement{{
											Key:      constant.ExternalIPLabel,
											Operator: corev1.NodeSelectorOpExists,
										}},
									},
								},
							},
						},
					},
					Tolerations: []corev1.Toleration{{
						Key:      constant.ControlPaneTaint,
						Operator: corev1.TolerationOpExists,
						Effect:   corev1.TaintEffectNoSchedule,
					}},
				},
			},
		},
	}
	if alreadyExists {
		if err = r.client.Update(ctx, deployment); err != nil {
			return reconcile.Result{}, fmt.Errorf("could not update Deployment: %+v", err)
		}
	} else {
		if err = r.client.Create(ctx, deployment); err != nil {
			return reconcile.Result{}, fmt.Errorf("could not create Deployment: %+v", err)
		}
	}

	if err = util.WaitForDeploymentComplete(r.client, deployment); err != nil {
		return reconcile.Result{}, fmt.Errorf("could not wait for Deployment to complete: %+v", err)
	}

	podList := &corev1.PodList{}
	if err = r.client.List(ctx, podList, &client.ListOptions{LabelSelector: labels.SelectorFromSet(map[string]string{"name": generateLoadBalancerName(svc)})}); err != nil {
		return reconcile.Result{}, fmt.Errorf("could not get podlist: %+v", err)
	}

	if len(podList.Items) == 0 {
		return reconcile.Result{}, fmt.Errorf("could not find pod: %+v", err)
	}

	pod := podList.Items[0]

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

func (r *reconcileService) ReconcileDelete(ctx context.Context, svc *corev1.Service) (reconcile.Result, error) {
	if util.ContainsString(svc.GetFinalizers(), constant.FinalizerName) {
		deployment := &appsv1.Deployment{}

		if err := r.client.Get(ctx, types.NamespacedName{Namespace: constant.Namespace, Name: generateLoadBalancerName(svc)}, deployment); err != nil {
			if errors.IsNotFound(err) {
				return reconcile.Result{}, nil
			}
			return reconcile.Result{}, fmt.Errorf("could not get deployment: %+v", err)
		}
		if err := r.client.Delete(ctx, deployment); err != nil {
			return reconcile.Result{}, fmt.Errorf("could not delete deployment: %+v", err)
		}

		controllerutil.RemoveFinalizer(svc, constant.FinalizerName)
		if err := r.client.Update(ctx, svc); err != nil {
			return reconcile.Result{}, fmt.Errorf("could not remove finalizer: %+v", err)
		}
	}
	return reconcile.Result{}, nil
}
