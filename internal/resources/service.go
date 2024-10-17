package resources

import (
	"context"
	"github.com/stakater/rhbk-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
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

const HttpPort = 8080
const HttpsPort = 8443

func GetTLSSecretName(cr *v1alpha1.Keycloak) string {
	return cr.Name + "-tls"
}

func GetSvcName(cr *v1alpha1.Keycloak) string {
	return cr.Name + "-svc"
}

func (s *RHBKService) Build() error {
	s.Resource = &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetSvcName(s.Keycloak),
			Namespace: s.Keycloak.Namespace,
			Labels: map[string]string{
				"app": "rhbk",
			},
			Annotations: map[string]string{
				"service.beta.openshift.io/serving-cert-secret-name": GetTLSSecretName(s.Keycloak),
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
	err := s.Build()

	if err != nil {
		return err
	}

	_, err = controllerutil.CreateOrUpdate(ctx, c, s.Resource, func() error { return nil })
	return err
}
