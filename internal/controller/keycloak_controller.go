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
	"github.com/stakater/rhbk-operator/internal/resources"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ssov1alpha1 "github.com/stakater/rhbk-operator/api/v1alpha1"
)

// KeycloakReconciler reconciles a Keycloak object
type KeycloakReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=sso.stakater.com,resources=keycloaks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=sso.stakater.com,resources=keycloaks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=sso.stakater.com,resources=keycloaks/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=StatefulSet,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=Service,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=route.openshift.io,resources=Route,verbs=get;list;watch;create;update;patch;delete

func (r *KeycloakReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx).WithName("keycloak-controller")

	cr := &ssov1alpha1.Keycloak{}
	if err := r.Get(ctx, req.NamespacedName, cr); client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	routeResource := resources.RHBKRoute{
		Keycloak: cr,
		Scheme:   r.Scheme,
	}

	err := routeResource.CreateOrUpdate(ctx, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	serviceResource := resources.RHBKService{
		Keycloak: cr,
		Scheme:   r.Scheme,
	}
	err = serviceResource.CreateOrUpdate(ctx, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	discoveryServiceResource := resources.RHBKDiscoveryService{
		Keycloak: cr,
		Scheme:   r.Scheme,
	}
	err = discoveryServiceResource.CreateOrUpdate(ctx, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	statefulSetResource := &resources.RHBKStatefulSet{
		Keycloak: cr,
		HostName: routeResource.Resource.Spec.Host,
		Scheme:   r.Scheme,
	}
	err = statefulSetResource.CreateOrUpdate(ctx, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KeycloakReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ssov1alpha1.Keycloak{}).
		Complete(r)
}
