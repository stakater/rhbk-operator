package resources

import (
	"context"
	"github.com/stakater/rhbk-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type RHBKService struct {
	Keycloak *v1alpha1.Keycloak
	Scheme   *runtime.Scheme
	Resource *v1.Service
}

const RHBKTlsSecretName = "rhbk-tls"
const HttpPort = 8080
const HttpsPort = 8443

func GetSvcName(cr *v1alpha1.Keycloak) string {
	return cr.Name + "-svc"
}

func (s *RHBKService) Build() error {
	s.Resource = &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Keycloak.Name,
			Namespace: s.Keycloak.Namespace,
			Labels: map[string]string{
				"app": "rhbk",
			},
			Annotations: map[string]string{
				"service.beta.openshift.io/serving-cert-secret-name": RHBKTlsSecretName,
			},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:     "http",
					Protocol: v1.ProtocolTCP,
					Port:     HttpPort,
					TargetPort: intstr.IntOrString{
						IntVal: HttpPort,
					},
				},
				{
					Name:     "https",
					Protocol: v1.ProtocolTCP,
					Port:     HttpsPort,
					TargetPort: intstr.IntOrString{
						IntVal: HttpsPort,
					},
				},
			},
			Selector: map[string]string{
				"app": "rhbk",
			},
		},
	}

	err := controllerutil.SetOwnerReference(s.Keycloak, s.Resource, s.Scheme)
	if err != nil {
		return err
	}

	return nil
}

func (s *RHBKService) CreateOrUpdate(ctx context.Context, c client.Client) error {
	err := c.Get(ctx, client.ObjectKey{
		Namespace: GetSvcName(s.Keycloak),
		Name:      s.Keycloak.Namespace,
	}, s.Resource)

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
