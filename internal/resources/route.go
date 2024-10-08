package resources

import (
	"context"
	v1 "github.com/openshift/api/route/v1"
	"github.com/stakater/rhbk-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RHBKRoute struct {
	Keycloak *v1alpha1.Keycloak
	Scheme   *runtime.Scheme
	Resource *v1.Route
}

func (s *RHBKRoute) Build() error {
	s.Resource = &v1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Keycloak.Name,
			Namespace: s.Keycloak.Namespace,
			Labels: map[string]string{
				"app": "rhbk",
			},
			Annotations: map[string]string{
				"route.openshift.io/destination-ca-certificate-secret": RHBKTlsSecretName,
				"route.openshift.io/termination":                       "reencrypt",
			},
		},
		Spec: v1.RouteSpec{
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
		},
	}

	return nil
}

func (s *RHBKRoute) CreateOrUpdate(ctx context.Context, c client.Client) error {
	existing := &v1.Route{}
	err := c.Get(ctx, client.ObjectKey{
		Namespace: GetSvcName(s.Keycloak),
		Name:      s.Keycloak.Namespace,
	}, existing)

	if err != nil {
		if errors.IsNotFound(err) {
			err = s.Build()
			if err != nil {
				return err
			}

			return c.Create(ctx, s.Resource)
		}

		return err
	}

	err = s.Build()
	if err != nil {
		return err
	}

	return c.Update(ctx, s.Resource)
}
