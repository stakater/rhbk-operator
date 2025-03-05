package monitoring

import (
	"context"
	"fmt"

	v1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stakater/rhbk-operator/api/v1alpha1"
	"github.com/stakater/rhbk-operator/internal/resources"
	"github.com/stakater/rhbk-operator/internal/resources/rhbk"
	v13 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type ServiceMonitorResource struct {
	Keycloak *v1alpha1.Keycloak
	Service  *v1.ServiceMonitor
	Scheme   *runtime.Scheme
}

func NewServiceMonitor(keycloak *v1alpha1.Keycloak, scheme *runtime.Scheme) *ServiceMonitorResource {
	return &ServiceMonitorResource{
		Keycloak: keycloak,
		Scheme:   scheme,
	}
}

func (m ServiceMonitorResource) mutateFn() error {
	defaultLabels := map[string]string{}
	resources.DecorateDefaultLabels(defaultLabels)

	m.Service.SetLabels(defaultLabels)
	m.Service.Spec = v1.ServiceMonitorSpec{
		Selector: v12.LabelSelector{
			MatchLabels: defaultLabels,
		},
		Endpoints: []v1.Endpoint{
			{
				Port:     rhbk.ManagementServicePortName,
				Path:     "/metrics",
				Scheme:   "https",
				Interval: "30s",
				TLSConfig: &v1.TLSConfig{
					SafeTLSConfig: v1.SafeTLSConfig{
						CA: v1.SecretOrConfigMap{
							Secret: &v13.SecretKeySelector{
								LocalObjectReference: v13.LocalObjectReference{
									Name: rhbk.GetTLSSecretName(m.Keycloak),
								},
								Key: "tls.crt",
							},
						},
						ServerName: &[]string{
							fmt.Sprintf("%s.%s.svc", rhbk.GetSvcName(m.Keycloak), m.Keycloak.Namespace),
						}[0],
					},
				},
			},
		},
	}

	return controllerutil.SetControllerReference(m.Keycloak, m.Service, m.Scheme)
}

func (m ServiceMonitorResource) CreateOrUpdate(ctx context.Context, c client.Client) error {
	m.Service = &v1.ServiceMonitor{
		ObjectMeta: v12.ObjectMeta{
			Name:      fmt.Sprintf("%s-servicemonitor", m.Keycloak.Name),
			Namespace: m.Keycloak.Namespace,
		},
	}

	_, err := controllerruntime.CreateOrUpdate(ctx, c, m.Service, m.mutateFn)
	return err
}
