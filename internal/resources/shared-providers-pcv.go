package resources

import (
	"context"
	"github.com/stakater/rhbk-operator/api/v1alpha1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type SharedPCV struct {
	Keycloak *v1alpha1.Keycloak
	Scheme   *runtime.Scheme
	Resource *v12.PersistentVolumeClaim
}

func GetSharedPVCName(cr *v1alpha1.Keycloak) string {
	return cr.Name + "-volume"
}

func (s *SharedPCV) Build() error {
	if !s.Resource.CreationTimestamp.IsZero() {
		return nil
	}

	s.Resource.ObjectMeta.Labels = GetDefaultLabels()
	s.Resource.Spec = v12.PersistentVolumeClaimSpec{
		AccessModes: []v12.PersistentVolumeAccessMode{
			v12.ReadWriteMany,
		},
		Resources: v12.VolumeResourceRequirements{
			Requests: v12.ResourceList{
				v12.ResourceStorage: resource.MustParse("1Gi"),
			},
		},
	}

	err := controllerutil.SetControllerReference(s.Keycloak, s.Resource, s.Scheme)
	if err != nil {
		return err
	}

	return nil
}

func (s *SharedPCV) CreateOrUpdate(ctx context.Context, c client.Client) error {
	s.Resource = &v12.PersistentVolumeClaim{
		ObjectMeta: v1.ObjectMeta{
			Name:      GetSharedPVCName(s.Keycloak),
			Namespace: s.Keycloak.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, c, s.Resource, s.Build)

	return err
}
