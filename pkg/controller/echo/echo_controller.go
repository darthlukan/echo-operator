package echo

import (
	"context"
	"fmt"
	"reflect"

	echov1alpha1 "github.com/darthlukan/echo-operator/pkg/apis/echo/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// DEFAULT_IMAGE is the application image we use
const DEFAULT_IMAGE string = "quay.io/btomlins/greet"

// DEFAULT_VERSION is the application image tag we use if one is not provided by the CustomResource.Spec.Version
// variable.
const DEFAULT_VERSION string = "latest"

var log = logf.Log.WithName("controller_echo")

// Add creates a new Echo Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileEcho{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("echo-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Echo
	err = c.Watch(&source.Kind{Type: &echov1alpha1.Echo{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Pods and requeue the owner Echo
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &echov1alpha1.Echo{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileEcho implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileEcho{}

// ReconcileEcho reconciles a Echo object
type ReconcileEcho struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads the state of the cluster for a Echo object and makes changes based on the state read
// and what is in the Echo.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileEcho) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Echo")

	// Fetch the Echo instance
	instance := &echov1alpha1.Echo{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Check if the Echo deployment already exists, if not create a new one
	// foundDep is a empty Deployment
	foundDep := &appsv1.Deployment{}
	// The Reconciler client queries for something named 'instance.Name' tracked by something of type foundDep (deployment)
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, foundDep)
	// if "err" is non-nil and contains the "IsNotFound" error, then we need to make a deployment for our Echo
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment for our Echo, pass in our CR instance of type Echo
		deployment := r.deploymentForEcho(instance)
		reqLogger.Info("Creating a new Deployment.", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		// err is the response from our Reconciler client attempting to make the deployment
		err = r.client.Create(context.TODO(), deployment)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Deployment.", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
			return reconcile.Result{}, err
		}
		// Deployment created successfully - return and requeue
		// We created our deployment, now we requeue so that our reconciler can continue with our logic
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		// There was no deployment or our error wasn't one of "Not Found"
		reqLogger.Error(err, "Failed to get Deployment.")
		return reconcile.Result{}, err
	}

	// Ensure the deployment size is the same as the spec in our CR instance
	size := instance.Spec.Replicas
	if *foundDep.Spec.Replicas != size {
		foundDep.Spec.Replicas = &size
		// If size doesn't match, update our deployment so it can match the number of pods in the namespace to our desired state
		err = r.client.Update(context.TODO(), foundDep)
		if err != nil {
			reqLogger.Error(err, "Failed to update Deployment.", "Deployment.Namespace", foundDep.Namespace, "Deployment.Name", foundDep.Name)
			return reconcile.Result{}, err
		}

		// Spec updated - return and requeue
		return reconcile.Result{Requeue: true}, nil
	}

	// Update the Echo status with the pod names
	// List the pods for this Echo's deployment
	podList := &corev1.PodList{}
	labelSelector := labels.SelectorFromSet(labelsForEcho(instance.Name))
	listOps := &client.ListOptions{Namespace: instance.Namespace, LabelSelector: labelSelector}
	// populate podList with pods matching listOps (our query params)
	err = r.client.List(context.TODO(), listOps, podList)
	if err != nil {
		reqLogger.Error(err, "Failed to list pods", "Echo.Namespace", instance.Namespace, "Echo.Name", instance.Name)
		return reconcile.Result{}, err
	}

	podNames := getPodNames(podList.Items)

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, instance.Status.Nodes) {
		instance.Status.Nodes = podNames
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Echo status")
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

// deploymentForEcho returns a Echo Deployment object
func (r *ReconcileEcho) deploymentForEcho(e *echov1alpha1.Echo) *appsv1.Deployment {
	replicas := e.Spec.Replicas
	labels := labelsForEcho(e.Name)
	// This is essentially a Golang representation of a YAML template
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.Name,
			Namespace: e.Spec.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metaForEcho(e, "pod"),
				Spec:       podSpecForEcho(e),
			},
		},
	}
	// Set Echo instance as the owner and controller
	controllerutil.SetControllerReference(e, deployment, r.scheme)
	return deployment
}

// labelsForEcho sets the labels baesd on the passed-in name
func labelsForEcho(name string) map[string]string {
	return map[string]string{"app": "echo", "echo_cr": name}
}

// metaForEcho returns a metadata object with values from the CR, t is a string representing the type this meta
// object is for. e.g. "pod" for a pod, "deployment" for deployment, etc.
func metaForEcho(cr *echov1alpha1.Echo, t string) metav1.ObjectMeta {
	labels := labelsForEcho(cr.Name)
	return metav1.ObjectMeta{
		Name:      cr.Name + "-" + t,
		Namespace: cr.Spec.Namespace,
		Labels:    labels,
	}
}

// podSpecForEcho returns a pod spec based on values in the Echo CR
func podSpecForEcho(cr *echov1alpha1.Echo) corev1.PodSpec {
	return corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:    cr.Name,
				Image:   fmt.Sprintf("%s:%s", DEFAULT_IMAGE, cr.Spec.Version),
				Command: []string{"/usr/bin/greet", cr.Spec.Message},
			},
		},
	}
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}
