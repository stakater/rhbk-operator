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

	"github.com/stakater/rhbk-operator/internal/resources/monitoring"

	v15 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stakater/rhbk-operator/internal/resources/rhbk"

	v12 "github.com/openshift/api/route/v1"
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
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type KeycloakReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=sso.stakater.com,resources=keycloaks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=sso.stakater.com,resources=keycloaks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=sso.stakater.com,resources=keycloaks/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update
//+kubebuilder:rbac:groups=route.openshift.io,resources=routes,verbs=get;list;watch;create;update
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;list;create;update;delete;watch

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

	serviceResource := rhbk.RHBKService{
		Keycloak: cr,
		Scheme:   r.Scheme,
	}
	err = serviceResource.CreateOrUpdate(ctx, r.Client)
	if err != nil {
		return r.HandleError(ctx, cr, err, "Service setup not ready")
	}

	routeResource := rhbk.RHBKRoute{
		Keycloak: cr,
		Scheme:   r.Scheme,
	}

	err = routeResource.CreateOrUpdate(ctx, r.Client)
	if err != nil {
		return r.HandleError(ctx, cr, err, "route setup not ready")
	}

	statefulSetResource := &rhbk.RHBKStatefulSet{
		Keycloak: cr,
		HostName: routeResource.Resource.Spec.Host,
		Scheme:   r.Scheme,
	}
	err = statefulSetResource.CreateOrUpdate(ctx, r.Client)
	if err != nil {
		return r.HandleError(ctx, cr, err, "Deployment setup not ready")
	}

	discoveryServiceResource := rhbk.RHBKDiscoveryService{
		Keycloak: cr,
		Scheme:   r.Scheme,
	}
	err = discoveryServiceResource.CreateOrUpdate(ctx, r.Client)
	if err != nil {
		return r.HandleError(ctx, cr, err, "Discovery service setup not ready")
	}

	serviceMonitorResource := monitoring.NewServiceMonitor(cr, r.Scheme)
	err = serviceMonitorResource.CreateOrUpdate(ctx, r.Client)
	if err != nil {
		return r.HandleError(ctx, cr, err, "Service monitor setup not ready")
	}

	if resources.IsStatefulSetReady(statefulSetResource.Resource) {
		return r.HandleSuccess(ctx, cr)
	} else {
		return r.HandleError(ctx, cr, nil, "Waiting for resources to be ready")
	}
}

func (r *KeycloakReconciler) HandleError(ctx context.Context, cr *ssov1alpha1.Keycloak, err error, msg string) (ctrl.Result, error) {
	original := cr.DeepCopy()

	if err != nil {
		cr.Status.Conditions.SetReady(v14.ConditionFalse, fmt.Sprintf("%s. %s", msg, err.Error()))
	} else {
		cr.Status.Conditions.SetReady(v14.ConditionFalse, msg)
	}
	return ctrl.Result{}, r.Status().Patch(ctx, cr, client.MergeFrom(original))
}

func (r *KeycloakReconciler) HandleSuccess(ctx context.Context, cr *ssov1alpha1.Keycloak) (ctrl.Result, error) {
	original := cr.DeepCopy()
	cr.Status.Conditions.SetReady(v14.ConditionTrue)
	return ctrl.Result{}, r.Status().Patch(ctx, cr, client.MergeFrom(original))
}

// SetupWithManager sets up the controller with the Manager.
func (r *KeycloakReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ssov1alpha1.Keycloak{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&v1.Service{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&v12.Route{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&v15.ServiceMonitor{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&v13.StatefulSet{}, builder.WithPredicates(predicate.Funcs{
			CreateFunc: func(e event.TypedCreateEvent[client.Object]) bool {
				return false
			},
			DeleteFunc: func(e event.TypedDeleteEvent[client.Object]) bool {
				return true
			},
			UpdateFunc: func(e event.TypedUpdateEvent[client.Object]) bool {
				old := e.ObjectOld.(*v13.StatefulSet)
				current := e.ObjectNew.(*v13.StatefulSet)

				return !resources.IsStatefulSetReady(old) && resources.IsStatefulSetReady(current)
			},
		})).
		Complete(r)
}
