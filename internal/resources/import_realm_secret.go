package resources

import (
	"fmt"
	"github.com/stakater/rhbk-operator/api/v1alpha1"
	"github.com/stakater/rhbk-operator/internal/constants"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type ImportRealmSecret struct {
	ImportCR *v1alpha1.KeycloakImport
	Scheme   *runtime.Scheme
	Resource *v1.Secret
}

func GetImportJobSecretName(cr *v1alpha1.KeycloakImport) string {
	return cr.Name
}

func GetImportJobSecretRealmName(cr *v1alpha1.KeycloakImport) string {
	return fmt.Sprintf("%s-realm.json", cr.Name)
}

func (s *ImportRealmSecret) Build() error {
	s.Resource = &v1.Secret{
		ObjectMeta: v12.ObjectMeta{
			Name:      GetImportJobSecretName(s.ImportCR),
			Namespace: s.ImportCR.Namespace,
			Labels: map[string]string{
				constants.RHBKRealmImportLabel: s.ImportCR.Name,
			},
		},
		Data: map[string][]byte{
			GetImportJobSecretRealmName(s.ImportCR): []byte(s.ImportCR.Spec.JSON),
		},
	}

	err := controllerutil.SetControllerReference(s.ImportCR, s.Resource, s.Scheme)
	if err != nil {
		return err
	}

	return nil
}
