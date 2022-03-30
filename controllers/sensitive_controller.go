/*
Copyright 2022.

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

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	testv1alpha1 "github.com/tkircsi/sensitive/api/v1alpha1"
)

// SensitiveReconciler reconciles a Sensitive object
type SensitiveReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=test.origoss.com,resources=sensitives,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=test.origoss.com,resources=sensitives/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=test.origoss.com,resources=sensitives/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Sensitive object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *SensitiveReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	log := r.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)

	// Fetch the Sensitive instance
	sen := &testv1alpha1.Sensitive{}
	err := r.Client.Get(ctx, req.NamespacedName, sen)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			sec, err := r.findSecret(ctx, req)
			if err == nil {
				err = r.deleteSecret(ctx, sec)
				if err != nil {
					log.Error(err, "Failed to delete secret")
				} else {
					r.Log.Info("Secret deleted", "Secret.Namespace", sec.Namespace, "Secret.Name", sec.Name)
				}
			} else {
				log.Error(err, "Cant find secret")
			}
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Cant get Sensitive")
		return reconcile.Result{}, err
	}

	// Check related Secret
	_, err = r.findSecret(ctx, req)
	if err != nil && errors.IsNotFound(err) {
		sec, err := r.createSecret(ctx, sen)
		if err != nil {
			log.Error(err, "Failed to create new Secret", "Secret.Namespace", sec.Namespace, "Secret.Name", sec.Name)
		}
	} else if err != nil {
		log.Error(err, "Failed to get secret")
	}

	return ctrl.Result{}, nil
}

func (r *SensitiveReconciler) deleteSecret(ctx context.Context, sec *corev1.Secret) error {
	/*	sec := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Namespace,
				Name:      req.Name,
			},
		}
	*/
	return r.Client.Delete(ctx, sec)
}

func (r *SensitiveReconciler) findSecret(ctx context.Context, req ctrl.Request) (*corev1.Secret, error) {
	found := &corev1.Secret{}
	err := r.Client.Get(ctx, types.NamespacedName{
		Name:      req.Name,
		Namespace: req.Namespace,
	}, found)
	return found, err
}

func (r *SensitiveReconciler) createSecret(ctx context.Context, instance *testv1alpha1.Sensitive) (*corev1.Secret, error) {

	immutable := true
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
		Immutable: &immutable,
		Type:      "Opaque",
		StringData: map[string]string{
			instance.Spec.Key: instance.Spec.Value,
		},
	}

	err := r.Client.Create(ctx, sec)
	if err != nil {
		r.Log.Error(err, "Failed to create new Secret", "Secret.Namespace", sec.Namespace, "Secret.Name", sec.Name)
	} else {
		r.Log.Info("Secret created", "Secret.Namespace", sec.Namespace, "Secret.Name", sec.Name)
	}
	return sec, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SensitiveReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&testv1alpha1.Sensitive{}).
		Complete(r)
}
