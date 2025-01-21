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
	v12 "github.com/openshift/api/route/v1"
	"github.com/redhat-cop/operator-utils/pkg/util/apis"
	ssov1alpha1 "github.com/stakater/rhbk-operator/api/v1alpha1"
	"github.com/stakater/rhbk-operator/internal/resources"
	v13 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v14 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/kustomize/kstatus/status"
)

type KeycloakReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=sso.stakater.com,resources=keycloaks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=sso.stakater.com,resources=keycloaks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=sso.stakater.com,resources=keycloaks/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update
//+kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update
//+kubebuilder:rbac:groups=route.openshift.io,resources=routes,verbs=get;list;watch;create;update

func (r *KeycloakReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("keycloak-controller")
	logger.Info("reconciling...")

	cr := &ssov1alpha1.Keycloak{}
	err := r.Get(ctx, req.NamespacedName, cr)

	if errors.IsNotFound(err) {
		return ctrl.Result{}, nil
	} else if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	serviceResource := resources.RHBKService{
		Keycloak: cr,
		Scheme:   r.Scheme,
	}
	err = serviceResource.CreateOrUpdate(ctx, r.Client)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	routeResource := resources.RHBKRoute{
		Keycloak: cr,
		Scheme:   r.Scheme,
	}

	err = routeResource.CreateOrUpdate(ctx, r.Client)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	discoveryServiceResource := resources.RHBKDiscoveryService{
		Keycloak: cr,
		Scheme:   r.Scheme,
	}
	err = discoveryServiceResource.CreateOrUpdate(ctx, r.Client)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	sharedPVC := &resources.SharedPCV{
		Keycloak: cr,
		Scheme:   r.Scheme,
	}

	err = sharedPVC.CreateOrUpdate(ctx, r.Client)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	statefulSetResource := &resources.RHBKStatefulSet{
		Keycloak: cr,
		HostName: routeResource.Resource.Spec.Host,
		Scheme:   r.Scheme,
	}
	err = statefulSetResource.CreateOrUpdate(ctx, r.Client)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	if err != nil {
		return ctrl.Result{}, err
	}

	println(resources.CheckStatus(statefulSetResource.Resource))

	if resources.CheckStatus(statefulSetResource.Resource) == status.CurrentStatus {
		cr.Status.Conditions = resources.AddOrReplaceCondition(v14.Condition{
			Type:               apis.ReconcileSuccess,
			Status:             v14.ConditionTrue,
			ObservedGeneration: statefulSetResource.Resource.Generation,
			LastTransitionTime: v14.Now(),
			Reason:             apis.ReconcileSuccessReason,
			Message:            "All resources are ready",
		}, cr.Status.Conditions)
	} else {
		cr.Status.Conditions = resources.AddOrReplaceCondition(v14.Condition{
			Type:               apis.ReconcileSuccess,
			Status:             v14.ConditionFalse,
			ObservedGeneration: statefulSetResource.Resource.Generation,
			LastTransitionTime: v14.Now(),
			Reason:             "Reconciling",
			Message:            "Waiting for resources to be ready",
		}, cr.Status.Conditions)
	}

	return ctrl.Result{}, r.Status().Update(ctx, cr)
}

// SetupWithManager sets up the controller with the Manager.
func (r *KeycloakReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ssov1alpha1.Keycloak{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&v1.Service{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&v12.Route{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&v13.StatefulSet{}).
		Owns(&v1.PersistentVolumeClaim{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}
