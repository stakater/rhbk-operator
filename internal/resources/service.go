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

const ManagementPort = 9000
const HttpsPort = 8443

func GetTLSSecretName(cr *v1alpha1.Keycloak) string {
	return cr.Name + "-tls"
}

func GetSvcName(cr *v1alpha1.Keycloak) string {
	return cr.Name + "-svc"
}

func (s *RHBKService) Build() error {
	s.Resource.Labels = map[string]string{
		"app": "rhbk",
	}
	s.Resource.Annotations = map[string]string{
		"service.beta.openshift.io/serving-cert-secret-name": GetTLSSecretName(s.Keycloak),
	}

	s.Resource.Spec = v1.ServiceSpec{
		Ports: []v1.ServicePort{
			{
				Name:     "management",
				Protocol: v1.ProtocolTCP,
				Port:     ManagementPort,
				TargetPort: intstr.IntOrString{
					IntVal: ManagementPort,
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
	}

	err := controllerutil.SetControllerReference(s.Keycloak, s.Resource, s.Scheme)
	if err != nil {
		return err
	}

	return nil
}

func (s *RHBKService) CreateOrUpdate(ctx context.Context, c client.Client) error {
	s.Resource = &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetSvcName(s.Keycloak),
			Namespace: s.Keycloak.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, c, s.Resource, s.Build)
	return err
}
