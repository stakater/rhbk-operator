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

type RHBKDiscoveryService struct {
	Keycloak *v1alpha1.Keycloak
	Scheme   *runtime.Scheme
	Resource *v1.Service
}

const DiscoveryPort = 7800

func GetDiscoverySvcName(cr *v1alpha1.Keycloak) string {
	return cr.Name + "-discovery"
}

func (s *RHBKDiscoveryService) Build() error {
	s.Resource = &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetDiscoverySvcName(s.Keycloak),
			Namespace: s.Keycloak.Namespace,
			Labels: map[string]string{
				"app": "rhbk",
			},
		},
		Spec: v1.ServiceSpec{
			ClusterIP:                "None",
			PublishNotReadyAddresses: true,
			Ports: []v1.ServicePort{
				{
					Name:     "http",
					Protocol: v1.ProtocolTCP,
					Port:     DiscoveryPort,
					TargetPort: intstr.IntOrString{
						IntVal: DiscoveryPort,
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

func (s *RHBKDiscoveryService) CreateOrUpdate(ctx context.Context, c client.Client) error {
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
