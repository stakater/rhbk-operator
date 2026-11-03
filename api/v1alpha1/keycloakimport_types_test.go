package v1alpha1

import (
	"testing"

	v1 "k8s.io/api/core/v1"
)

func TestHasSecretReference(t *testing.T) {
	tests := []struct {
		name          string
		substitutions []SecretOptionVar
		secretName    string
		want          bool
	}{
		{
			name:          "no substitutions",
			substitutions: nil,
			secretName:    "any",
			want:          false,
		},
		{
			name: "matching secret reference",
			substitutions: []SecretOptionVar{
				{
					Name: "VAR",
					Secret: &v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{Name: "realm-secret"},
						Key:                  "key",
					},
				},
			},
			secretName: "realm-secret",
			want:       true,
		},
		{
			name: "non-matching secret reference",
			substitutions: []SecretOptionVar{
				{
					Name: "VAR",
					Secret: &v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{Name: "other-secret"},
						Key:                  "key",
					},
				},
			},
			secretName: "realm-secret",
			want:       false,
		},
		{
			name: "value-only substitution must not panic",
			substitutions: []SecretOptionVar{
				{Name: "DISPLAY_NAME", Value: "My Realm"},
			},
			secretName: "realm-secret",
			want:       false,
		},
		{
			name: "mixed value-only and matching secret reference",
			substitutions: []SecretOptionVar{
				{Name: "DISPLAY_NAME", Value: "My Realm"},
				{
					Name: "VAR",
					Secret: &v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{Name: "realm-secret"},
						Key:                  "key",
					},
				},
			},
			secretName: "realm-secret",
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := &KeycloakImportSpec{Substitutions: tt.substitutions}
			if got := spec.HasSecretReference(tt.secretName); got != tt.want {
				t.Errorf("HasSecretReference(%q) = %v, want %v", tt.secretName, got, tt.want)
			}
		})
	}
}
