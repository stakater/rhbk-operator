package resources

import (
	"bytes"
	"fmt"
	"github.com/stakater/rhbk-operator/api/v1alpha1"
	"github.com/stakater/rhbk-operator/internal/constants"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
	"text/template"
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

func (s *ImportRealmSecret) Build(substitutions map[string]string) error {
	realm, err := expandTemplate(s.ImportCR.Spec.JSON, substitutions)
	if err != nil {
		return err
	}

	s.Resource = &v1.Secret{
		ObjectMeta: v12.ObjectMeta{
			Name:      GetImportJobSecretName(s.ImportCR),
			Namespace: s.ImportCR.Namespace,
			Labels: map[string]string{
				constants.RHBKRealmImportLabel:     s.ImportCR.Name,
				constants.RHBKWatchedResourceLabel: strconv.FormatBool(true),
			},
		},
		Data: map[string][]byte{
			GetImportJobSecretRealmName(s.ImportCR): realm,
		},
	}

	err = controllerutil.SetControllerReference(s.ImportCR, s.Resource, s.Scheme)
	if err != nil {
		return err
	}

	return nil
}

func expandTemplate(t string, substitutions map[string]string) ([]byte, error) {
	tpl, err := template.New("Secret replacer").Option("missingkey=error").Delims("%", "%").Parse(t)
	if err != nil {
		return nil, err
	}

	var results bytes.Buffer
	if err := tpl.Execute(&results, substitutions); err != nil {
		return nil, err
	}

	return results.Bytes(), nil
}
