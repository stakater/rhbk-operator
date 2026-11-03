package realm

import (
	"context"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stakater/rhbk-operator/api/v1alpha1"
)

func TestImportRealmSecretCreateOrUpdate(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to build scheme: %v", err)
	}
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to build scheme: %v", err)
	}

	referencedSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "realm-secret",
			Namespace: "team-a",
		},
		Data: map[string][]byte{
			"client-secret": []byte("s3cr3t"),
		},
	}

	tests := []struct {
		name          string
		substitutions []v1alpha1.SecretOptionVar
		json          string
		wantRendered  string
	}{
		{
			name: "secret reference substitution",
			substitutions: []v1alpha1.SecretOptionVar{
				{
					Name: "CLIENT_SECRET",
					Secret: &v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{Name: "realm-secret"},
						Key:                  "client-secret",
					},
				},
			},
			json:         `{"secret": "%.CLIENT_SECRET%"}`,
			wantRendered: `{"secret": "s3cr3t"}`,
		},
		{
			name: "value-only substitution must not panic and must be used",
			substitutions: []v1alpha1.SecretOptionVar{
				{Name: "DISPLAY_NAME", Value: "Team A"},
			},
			json:         `{"displayName": "%.DISPLAY_NAME%"}`,
			wantRendered: `{"displayName": "Team A"}`,
		},
		{
			name: "mixed value and secret substitutions",
			substitutions: []v1alpha1.SecretOptionVar{
				{Name: "DISPLAY_NAME", Value: "Team A"},
				{
					Name: "CLIENT_SECRET",
					Secret: &v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{Name: "realm-secret"},
						Key:                  "client-secret",
					},
				},
			},
			json:         `{"displayName": "%.DISPLAY_NAME%", "secret": "%.CLIENT_SECRET%"}`,
			wantRendered: `{"displayName": "Team A", "secret": "s3cr3t"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(referencedSecret).Build()

			importCR := &v1alpha1.KeycloakImport{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "team-a-realm",
					Namespace: "team-a",
				},
				Spec: v1alpha1.KeycloakImportSpec{
					KeycloakInstance: v1alpha1.KeycloakInstance{
						Name:      "keycloak",
						Namespace: "team-a",
					},
					JSON:          tt.json,
					Substitutions: tt.substitutions,
				},
			}

			s := &ImportRealmSecret{
				ImportCR: importCR,
				Scheme:   scheme,
			}

			if err := s.CreateOrUpdate(context.Background(), c); err != nil {
				t.Fatalf("CreateOrUpdate failed: %v", err)
			}

			rendered := string(s.Resource.Data[GetImportJobSecretRealmName(importCR)])
			if rendered != tt.wantRendered {
				t.Errorf("rendered realm = %q, want %q", rendered, tt.wantRendered)
			}
		})
	}
}
