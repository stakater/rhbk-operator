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
	ssov1alpha1 "github.com/stakater/rhbk-operator/api/v1alpha1"
	"github.com/stakater/rhbk-operator/internal/resources"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/batch/v1"
	v13 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"
)

// KeycloakImportReconciler reconciles a KeycloakImport object
type KeycloakImportReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=sso.stakater.com,resources=keycloakimports,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=sso.stakater.com,resources=keycloakimports/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=sso.stakater.com,resources=keycloakimports/finalizers,verbs=update
//+kubebuilder:rbac:groups=sso.stakater.com,resources=keycloaks,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=Secret,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=StatefulSet,verbs=get;

func (r *KeycloakImportReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	realmCR := &ssov1alpha1.KeycloakImport{}
	err := r.Get(ctx, req.NamespacedName, realmCR)

	if errors.IsNotFound(err) {
		return ctrl.Result{}, nil
	} else if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	keycloak := &ssov1alpha1.Keycloak{}
	err = r.Get(ctx, client.ObjectKey{
		Namespace: realmCR.Spec.KeycloakInstance.Namespace,
		Name:      realmCR.Spec.KeycloakInstance.Name,
	}, keycloak)

	if err != nil {
		logger.Info("RHBK resource is does not exists")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	statefulSet := &v1.StatefulSet{}
	err = r.Get(ctx, client.ObjectKey{
		Name:      resources.GetStatefulSetName(keycloak),
		Namespace: keycloak.Namespace,
	}, statefulSet)

	if err != nil {
		logger.Info("RHBK deployment is not yet ready")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Fetch substitutions
	substitutions := make(map[string]string)
	for _, s := range realmCR.Spec.Substitutions {
		secret := &v13.Secret{}
		err = r.Get(ctx, client.ObjectKey{
			Name:      s.Secret.Name,
			Namespace: realmCR.Namespace,
		}, secret)

		if err != nil {
			logger.Error(err, "error fetching secret")
			return ctrl.Result{}, err
		}

		substitutions[s.Name] = string(secret.Data[s.Secret.Key])
	}

	realmSecret := &v13.Secret{}
	err = r.Get(ctx, client.ObjectKey{
		Name:      resources.GetImportJobSecretName(realmCR),
		Namespace: statefulSet.Namespace,
	}, realmSecret)

	if err != nil {
		if errors.IsNotFound(err) {
			sr := &resources.ImportRealmSecret{
				ImportCR: realmCR,
				Scheme:   r.Scheme,
			}

			err = sr.Build(substitutions)
			if err != nil {
				return ctrl.Result{}, err
			}

			err = r.Create(ctx, sr.Resource)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, err
	}

	job := &v12.Job{}
	err = r.Get(ctx, client.ObjectKey{
		Name:      resources.GetImportJobName(realmCR),
		Namespace: statefulSet.Namespace,
	}, job)

	if err != nil {
		if errors.IsNotFound(err) {
			job := &resources.ImportJob{
				ImportCR:    realmCR,
				Scheme:      r.Scheme,
				StatefulSet: statefulSet,
			}

			err := job.Build()
			if err != nil {
				return ctrl.Result{}, err
			}

			err = r.Create(ctx, job.Job)
			if err != nil {
				return ctrl.Result{}, err
			}
		} else {
			return ctrl.Result{}, err
		}
	}

	if resources.IsJobCompleted(job) {
		return ctrl.Result{}, r.RolloutChanges(ctx, statefulSet)
	}

	return ctrl.Result{}, nil
}

func (r *KeycloakImportReconciler) RolloutChanges(ctx context.Context, statefulSet *v1.StatefulSet) error {
	if statefulSet.Spec.Template.Annotations == nil {
		statefulSet.Spec.Template.Annotations = make(map[string]string)
	}
	statefulSet.Spec.Template.Annotations["statefulset.kubernetes.io/rollout"] = time.Now().Format(time.RFC3339)
	return r.Update(ctx, statefulSet)
}

// SetupWithManager sets up the controller with the Manager.
func (r *KeycloakImportReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ssov1alpha1.KeycloakImport{}).
		Owns(&v13.Secret{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&v12.Job{}, builder.WithPredicates(predicate.Funcs{
			UpdateFunc: func(e event.TypedUpdateEvent[client.Object]) bool {
				old := e.ObjectOld.(*v12.Job)
				recent := e.ObjectNew.(*v12.Job)

				return !resources.IsJobCompleted(old) && resources.IsJobCompleted(recent)
			},
		})).
		Watches(&ssov1alpha1.Keycloak{}, &handler.EnqueueRequestForObject{},
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}
