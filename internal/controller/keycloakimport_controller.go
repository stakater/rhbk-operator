/*
Copyright 2024.

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

	"github.com/go-logr/logr"
	ssov1alpha1 "github.com/stakater/rhbk-operator/api/v1alpha1"
	"github.com/stakater/rhbk-operator/internal/constants"
	"github.com/stakater/rhbk-operator/internal/resources"
	v1 "k8s.io/api/apps/v1"
	v14 "k8s.io/api/batch/v1"
	v13 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// KeycloakImportReconciler reconciles a KeycloakImport object
type KeycloakImportReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

//+kubebuilder:rbac:groups=sso.stakater.com,resources=keycloakimports,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=sso.stakater.com,resources=keycloakimports/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=sso.stakater.com,resources=keycloakimports/finalizers,verbs=update
//+kubebuilder:rbac:groups=sso.stakater.com,resources=keycloaks,verbs=get;list;watch
//+kubebuilder:rbac:groups=sso.stakater.com,resources=keycloaks/status,verbs=get
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;update

func (r *KeycloakImportReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx)
	r.logger.Info("reconciling...")

	cr := &ssov1alpha1.KeycloakImport{}
	err := r.Get(ctx, req.NamespacedName, cr)

	if errors.IsNotFound(err) {
		return ctrl.Result{}, nil
	} else if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	keycloak := &ssov1alpha1.Keycloak{}
	err = r.Get(ctx, client.ObjectKey{
		Namespace: cr.Spec.KeycloakInstance.Namespace,
		Name:      cr.Spec.KeycloakInstance.Name,
	}, keycloak)

	if err != nil {
		return r.HandleError(ctx, cr, err, "Failed to fetch RHBK instance")
	}

	// Don't do anything if rhbk instance is not ready
	if !keycloak.Status.Conditions.IsReady() {
		return r.HandleError(ctx, cr, err, "RHBK instance not ready")
	}

	statefulSet := &v1.StatefulSet{}
	err = r.Get(ctx, client.ObjectKey{
		Name:      resources.GetStatefulSetName(keycloak),
		Namespace: keycloak.Namespace,
	}, statefulSet)

	if err != nil {
		return r.HandleError(ctx, cr, err, "RHBK deployment not ready")
	}

	importSecret := &resources.ImportRealmSecret{
		ImportCR: cr,
		Scheme:   r.Scheme,
	}
	err = importSecret.CreateOrUpdate(ctx, r.Client)
	if err != nil {
		return r.HandleError(ctx, cr, err, "Realm secret not ready")
	}

	jobs := &v14.JobList{}
	err = r.List(ctx, jobs, client.InNamespace(statefulSet.Namespace), client.MatchingLabelsSelector{
		Selector: labels.SelectorFromSet(map[string]string{
			constants.RHBKRealmImportLabel: cr.Name,
		}),
	})

	if err != nil {
		return r.HandleError(ctx, cr, err, "Failed to fetch import job")
	}

	var found *v14.Job
	for _, job := range jobs.Items {
		if job.Labels[constants.RHBKRealmImportRevisionLabel] == importSecret.Resource.ResourceVersion {
			found = &job
			break
		} else {
			err = r.Delete(ctx, &job, []client.DeleteOption{
				client.PropagationPolicy(v12.DeletePropagationForeground),
			}...)
			if err != nil {
				return r.HandleError(ctx, cr, err, "Failed to delete old job")
			}
		}
	}

	// If no job found create job and wait for next reconcile when job is completed
	if found == nil {
		importJob, err := resources.Build(cr, statefulSet, importSecret.Resource.ResourceVersion, r.Scheme)
		if err != nil {
			return r.HandleError(ctx, cr, err, "Failed to build import job")
		}

		err = r.Create(ctx, importJob)
		if err != nil {
			return r.HandleError(ctx, cr, err, "Failed to create import job")
		}

		return r.HandleError(ctx, cr, err, "Wait for new import job to be ready")
	}

	if !resources.MatchSet(statefulSet.Spec.Template.Annotations, map[string]string{
		"statefulset.kubernetes.io/rollout": importSecret.Resource.ResourceVersion,
	}) && resources.IsJobCompleted(found) {
		return ctrl.Result{}, r.RolloutChanges(ctx, statefulSet, importSecret.Resource.ResourceVersion)
	}

	return r.HandleSuccess(ctx, cr)
}

func (r *KeycloakImportReconciler) HandleError(ctx context.Context, cr *ssov1alpha1.KeycloakImport, err error, msg string) (ctrl.Result, error) {
	original := cr.DeepCopy()
	if err != nil {
		cr.Status.Conditions.SetReady(v12.ConditionFalse, fmt.Sprintf("%s. %s", msg, err.Error()))
	} else {
		cr.Status.Conditions.SetReady(v12.ConditionFalse, msg)
	}

	return ctrl.Result{}, r.Status().Patch(ctx, cr, client.MergeFrom(original))
}

func (r *KeycloakImportReconciler) HandleSuccess(ctx context.Context, cr *ssov1alpha1.KeycloakImport) (ctrl.Result, error) {
	original := cr.DeepCopy()
	cr.Status.Conditions.SetReady(v12.ConditionTrue)
	return ctrl.Result{}, r.Status().Patch(ctx, cr, client.MergeFrom(original))
}

func (r *KeycloakImportReconciler) RolloutChanges(ctx context.Context, statefulSet *v1.StatefulSet, revision string) error {
	original := statefulSet.DeepCopy()
	if statefulSet.Spec.Template.Annotations == nil {
		statefulSet.Spec.Template.Annotations = make(map[string]string)
	}
	statefulSet.Spec.Template.Annotations["statefulset.kubernetes.io/rollout"] = revision

	return r.Patch(ctx, statefulSet, client.MergeFrom(original))
}

// SetupWithManager sets up the controller with the Manager.
func (r *KeycloakImportReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ssov1alpha1.KeycloakImport{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&v14.Job{}, builder.WithPredicates(predicate.Funcs{
			CreateFunc: func(e event.TypedCreateEvent[client.Object]) bool {
				return false
			},
			DeleteFunc: func(e event.TypedDeleteEvent[client.Object]) bool {
				return false
			},
			UpdateFunc: func(e event.TypedUpdateEvent[client.Object]) bool {
				old := e.ObjectOld.(*v14.Job)
				current := e.ObjectNew.(*v14.Job)

				return !resources.IsJobCompleted(old) && resources.IsJobCompleted(current)
			},
		})).
		Watches(&ssov1alpha1.Keycloak{}, handler.EnqueueRequestsFromMapFunc(r.handleRHBKChanged)).
		Watches(&v13.Secret{}, handler.EnqueueRequestsFromMapFunc(r.handleSecretChanged)).
		Complete(r)
}

func (r *KeycloakImportReconciler) handleSecretChanged(ctx context.Context, object client.Object) []reconcile.Request {
	secret := object.(*v13.Secret)
	imports := &ssov1alpha1.KeycloakImportList{}
	err := r.List(ctx, imports)
	if err != nil {
		r.logger.Error(err, "unable to list RHBK instances")
		return nil
	}

	var requests []reconcile.Request
	for _, cr := range imports.Items {
		if resources.MatchSet(secret.GetLabels(), map[string]string{
			constants.RHBKRealmImportLabel: cr.Name,
		}) || cr.Spec.HasSecretReference(secret.Name) {
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Namespace: cr.Namespace,
					Name:      cr.Name,
				},
			})
		}
	}

	return requests
}

func (r *KeycloakImportReconciler) handleRHBKChanged(ctx context.Context, object client.Object) []reconcile.Request {
	rhbk := object.(*ssov1alpha1.Keycloak)
	imports := &ssov1alpha1.KeycloakImportList{}
	err := r.List(ctx, imports)
	if err != nil {
		r.logger.Error(err, "unable to list RHBK instances")
		return nil
	}

	var requests []reconcile.Request
	for _, cr := range imports.Items {
		if cr.Spec.KeycloakInstance.Name == rhbk.Name && cr.Spec.KeycloakInstance.Namespace == rhbk.Namespace {
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Namespace: cr.Namespace,
					Name:      cr.Name,
				},
			})
		}
	}

	return requests
}
