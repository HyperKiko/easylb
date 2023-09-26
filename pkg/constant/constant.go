package constant

const (
	ExternalIPLabel                    = "easylb.hyperkiko.github.io/external-ip"
	ExcludeNodeLabel                   = "easylb.hyperkiko.github.io/exclude-node"

	ManagedAnnotation                  = "easylb.hyperkiko.github.io/managed"
	LoadBalancerForNamespaceAnnotation = "easylb.hyperkiko.github.io/load-balancer-for-namespace"
	LoadBalancerForNameAnnotation      = "easylb.hyperkiko.github.io/load-balancer-for-name"

	ControlPaneTaint                   = "node-role.kubernetes.io/control-plane"
	
	FinalizerName                      = "easylb.hyperkiko.github.io/finalizer"
	
	Namespace 						   = "easylb-system"

	LoadBalancerImage 				   = "kikocodes/easylb-loadbalancer"
	EnableIPForwardImage 			   = "kikocodes/enable-ip-forward"
)
