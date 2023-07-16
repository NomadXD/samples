/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	podrunnerv1alpha1 "github.com/NomadXD/samples/k8s-operator-kube-builder/api/v1alpha1"
)

// PodRunnerReconciler reconciles a PodRunner object
type PodRunnerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=podrunner.nomadxd.io,resources=podrunners,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=podrunner.nomadxd.io,resources=podrunners/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=podrunner.nomadxd.io,resources=podrunners/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PodRunner object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *PodRunnerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	podRunner := &podrunnerv1alpha1.PodRunner{}
	err := r.Get(ctx, req.NamespacedName, podRunner)
	if err != nil {
		// Error reading the PodRunner instance, requeue the request
		logger.Error(err, "Failed to get PodRunner")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Create a Pod based on the PodRunner specification
	podRunnerPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podRunner.Spec.PodName,
			Namespace: podRunner.Spec.Namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            podRunner.Spec.PodName,
					Image:           podRunner.Spec.ImageName,
					ImagePullPolicy: corev1.PullAlways,
				},
			},
		},
	}

	err = ctrl.SetControllerReference(podRunner, podRunnerPod, r.Scheme)
	if err != nil {
		logger.Error(err, "Failed to set controller reference for Nginx Pod")
		return ctrl.Result{}, err
	}

	// Check if the Pod already exists
	foundPod := &corev1.Pod{}
	err = r.Get(ctx, types.NamespacedName{Name: podRunner.Spec.PodName, Namespace: podRunner.Spec.Namespace}, foundPod)
	if err != nil && errors.IsNotFound(err) {
		// Create the Pod
		err = r.Create(ctx, podRunnerPod)
		if err != nil {
			logger.Error(err, "Failed to create Pod")
			return ctrl.Result{}, err
		}
		logger.Info("Pod created")
		return ctrl.Result{}, nil
	} else if err != nil {
		logger.Error(err, "Failed to get Pod")
		return ctrl.Result{}, err
	}

	// Pod already exists, do nothing
	logger.Info("Pod already exists")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodRunnerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&podrunnerv1alpha1.PodRunner{}).
		Complete(r)
}
