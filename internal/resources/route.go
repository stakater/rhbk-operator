package resources

import (
	"context"
	v1 "github.com/openshift/api/route/v1"
	"github.com/stakater/rhbk-operator/api/v1alpha1"
	"github.com/stakater/rhbk-operator/internal/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
)

type RHBKRoute struct {
	Keycloak *v1alpha1.Keycloak
	Scheme   *runtime.Scheme
	Resource *v1.Route
}

func (s *RHBKRoute) Build() error {
	s.Resource.Labels = map[string]string{
		"app":                  "rhbk",
		constants.RHBKAppLabel: strconv.FormatBool(true),
	}

	s.Resource.Spec = v1.RouteSpec{
		To: v1.RouteTargetReference{
			Kind: "Service",
			Name: GetSvcName(s.Keycloak),
		},
		Port: &v1.RoutePort{
			TargetPort: intstr.FromString("https"),
		},
		TLS: &v1.TLSConfig{
			Termination:                   v1.TLSTerminationReencrypt,
			InsecureEdgeTerminationPolicy: v1.InsecureEdgeTerminationPolicyRedirect,
		},
	}

	err := controllerutil.SetControllerReference(s.Keycloak, s.Resource, s.Scheme)
	if err != nil {
		return err
	}

	return nil
}

func (s *RHBKRoute) CreateOrUpdate(ctx context.Context, c client.Client) error {
	s.Resource = &v1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Keycloak.Name,
			Namespace: s.Keycloak.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, c, s.Resource, s.Build)

	return err
}
