package main

import (
	"os"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func init() {
	log.SetLogger(zap.New())
}

func main() {
	entryLog := log.Log.WithName("entrypoint")

	entryLog.Info("setting up manager")
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{})
	if err != nil {
		entryLog.Error(err, "unable to set up overall controller manager")
		os.Exit(1)
	}

	entryLog.Info("Setting up pod controller")
	podCtrl, err := controller.New("easylb-pod-controller", mgr, controller.Options{
		Reconciler: &reconcilePod{client: mgr.GetClient()},
	})
	if err != nil {
		entryLog.Error(err, "unable to set up the easylb pod controller")
		os.Exit(1)
	}

	if err := podCtrl.Watch(source.Kind(mgr.GetCache(), &corev1.Pod{}), &handler.EnqueueRequestForObject{}); err != nil {
		entryLog.Error(err, "unable to watch Pods")
		os.Exit(1)
	}

	entryLog.Info("Setting up node controller")
	nodeCtrl, err := controller.New("easylb-node-controller", mgr, controller.Options{
		Reconciler: &reconcileNode{client: mgr.GetClient()},
	})
	if err != nil {
		entryLog.Error(err, "unable to set up the easylb node controller")
		os.Exit(1)
	}

	if err := nodeCtrl.Watch(source.Kind(mgr.GetCache(), &corev1.Node{}), &handler.EnqueueRequestForObject{}); err != nil {
		entryLog.Error(err, "unable to watch Nodes")
		os.Exit(1)
	}

	entryLog.Info("Setting up service controller")
	svcCtrl, err := controller.New("easylb-svc-controller", mgr, controller.Options{
		Reconciler: &reconcileService{client: mgr.GetClient()},
	})
	if err != nil {
		entryLog.Error(err, "unable to set up the easylb service controller")
		os.Exit(1)
	}

	if err := svcCtrl.Watch(source.Kind(mgr.GetCache(), &corev1.Service{}), &handler.EnqueueRequestForObject{}); err != nil {
		entryLog.Error(err, "unable to watch Services")
		os.Exit(1)
	}

	entryLog.Info("starting manager")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		entryLog.Error(err, "unable to run manager")
		os.Exit(1)
	}
}
