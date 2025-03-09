/*
Copyright 2025.

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

package controller

import (
	"context"
	"fmt"
	"slices"

	es "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ImagePullSecretReconciler reconciles a ImagePullSecret
type ImagePullSecretReconciler struct {
	client.Client
	Scheme                 *runtime.Scheme
	TriggerSecretName      string
	DesiredImagePullSecret es.ExternalSecret
}

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=pods/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Pod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *ImagePullSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var pod corev1.Pod
	err := r.Get(ctx, req.NamespacedName, &pod)
	// we perform reconciliation if the pod was deleted (i.e. errors.IsNotFound(err) == true),
	// thus excluding the case
	if err != nil && !errors.IsNotFound(err) {
		logger.Error(err, "unable to get pod", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}
	// do nothing on the beginning of pod deletion
	if !pod.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}
	// check whether the pod uses r.TriggerSecretName as an imagePullSecret
	matchesTargetSecretName := func(obj corev1.LocalObjectReference) bool {
		return obj.Name == r.TriggerSecretName
	}
	if slices.ContainsFunc(pod.Spec.ImagePullSecrets, matchesTargetSecretName) {
		err = r.reconcileExternalSecrets(ctx, pod.Namespace)
		if err != nil {
			logger.Error(err, "unable to reconcile external secrets")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ImagePullSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Named("pod").
		Complete(r)
}

func (r *ImagePullSecretReconciler) reconcileExternalSecrets(ctx context.Context, namespace string) error {
	// check if the namespace requires r.targetSecretName secret
	pods := &corev1.PodList{}
	err := r.Client.List(ctx, pods, &client.ListOptions{Namespace: namespace})
	if err != nil {
		return fmt.Errorf("listing pods in %s namespace: %v", namespace, err)
	}
	usesTargetSecret := func(pod corev1.Pod) bool {
		for _, obj := range pod.Spec.ImagePullSecrets {
			if obj.Name == r.TriggerSecretName {
				return true
			}
		}
		return false
	}
	targetSecretRequired := slices.ContainsFunc(pods.Items, usesTargetSecret)

	// check if the namespace has r.targetSecretName secret
	secrets := &corev1.SecretList{}
	err = r.Client.List(ctx, secrets, &client.ListOptions{Namespace: namespace})
	if err != nil {
		return fmt.Errorf("listing secrets in %s namespace: %v", namespace, err)
	}
	targetSecretExists := false
	for _, secret := range secrets.Items {
		if secret.Name == r.TriggerSecretName {
			targetSecretExists = true
		}
	}

	// performs creation/deletion of the ExternalSecret
	// what happens with the child Secret depends on the ExternalSecret's spec.target.creationPolicy
	if targetSecretRequired && !targetSecretExists {
		desiredExternalSecret := r.DesiredImagePullSecret
		desiredExternalSecret.SetNamespace(namespace)
		err := r.Client.Create(ctx, &desiredExternalSecret)
		if err != nil {
			return fmt.Errorf("creating ExternalSecret: %v", err)
		}
		return nil
	}

	if !targetSecretRequired && targetSecretExists {
		desiredExternalSecret := r.DesiredImagePullSecret
		desiredExternalSecret.SetNamespace(namespace)
		err := r.Client.Delete(ctx, &desiredExternalSecret)
		if err != nil {
			return fmt.Errorf("deleting ExternalSecret: %v", err)
		}
	}

	return nil
}
