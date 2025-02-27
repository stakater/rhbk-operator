package resources

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"text/template"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/stakater/rhbk-operator/internal/constants"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/stakater/rhbk-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type ImportRealmSecret struct {
	ImportCR      *v1alpha1.KeycloakImport
	Resource      *v1.Secret
	Scheme        *runtime.Scheme
	substitutions map[string]string
}

func GetImportJobSecretName(cr *v1alpha1.KeycloakImport) string {
	return cr.Name
}

func GetImportJobSecretRealmName(cr *v1alpha1.KeycloakImport) string {
	return fmt.Sprintf("%s-realm.json", cr.Name)
}

func (s *ImportRealmSecret) CreateOrUpdate(ctx context.Context, c client.Client) error {
	s.Resource = &v1.Secret{
		ObjectMeta: v12.ObjectMeta{
			Name:      GetImportJobSecretName(s.ImportCR),
			Namespace: s.ImportCR.Spec.KeycloakInstance.Namespace,
		},
	}

	// Fetch substitutions
	s.substitutions = make(map[string]string)
	for _, sub := range s.ImportCR.Spec.Substitutions {
		secret := &v1.Secret{}
		err := c.Get(ctx, client.ObjectKey{
			Name:      sub.Secret.Name,
			Namespace: s.ImportCR.Namespace,
		}, secret)

		if err != nil {
			return err
		}

		s.substitutions[sub.Name] = string(secret.Data[sub.Secret.Key])
	}

	_, err := controllerutil.CreateOrUpdate(ctx, c, s.Resource, s.MutateFn)
	return err
}

func (s *ImportRealmSecret) MutateFn() error {
	realm, err := expandTemplate(s.ImportCR.Spec.JSON, s.substitutions)
	if err != nil {
		return err
	}

	ownerLabels := GetOwnerLabels(s.ImportCR.Name, s.ImportCR.Namespace)
	ownerLabels[constants.RHBKWatchedResourceLabel] = strconv.FormatBool(true)
	s.Resource.Labels = ownerLabels

	s.Resource.Data = map[string][]byte{
		GetImportJobSecretRealmName(s.ImportCR): realm,
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

func FindAllImportSecrets(ctx context.Context, cr *v1alpha1.KeycloakImport, c client.Client) (*v1.SecretList, error) {
	kcNamespace := cr.Spec.KeycloakInstance.Namespace
	ownerLabels := labels.SelectorFromSet(GetOwnerLabels(cr.Name, cr.Namespace))

	secrets := &v1.SecretList{}
	err := c.List(ctx, secrets, client.InNamespace(kcNamespace), client.MatchingLabelsSelector{
		Selector: ownerLabels,
	})

	if err != nil {
		return nil, err
	}

	return secrets, nil
}
