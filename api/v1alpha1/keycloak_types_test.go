package v1alpha1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAllowsImportFrom(t *testing.T) {
	tests := []struct {
		name            string
		policy          *RealmImportPolicy
		sourceNamespace string
		want            bool
	}{
		{
			name:            "same namespace is always allowed without a policy",
			policy:          nil,
			sourceNamespace: "team-a",
			want:            true,
		},
		{
			name:            "cross-namespace is denied without a policy",
			policy:          nil,
			sourceNamespace: "team-b",
			want:            false,
		},
		{
			name:            "cross-namespace is denied with an empty policy",
			policy:          &RealmImportPolicy{},
			sourceNamespace: "team-b",
			want:            false,
		},
		{
			name:            "listed namespace is allowed",
			policy:          &RealmImportPolicy{AllowedNamespaces: []string{"team-b", "team-c"}},
			sourceNamespace: "team-b",
			want:            true,
		},
		{
			name:            "unlisted namespace is denied",
			policy:          &RealmImportPolicy{AllowedNamespaces: []string{"team-c"}},
			sourceNamespace: "team-b",
			want:            false,
		},
		{
			name:            "wildcard allows any namespace",
			policy:          &RealmImportPolicy{AllowedNamespaces: []string{"*"}},
			sourceNamespace: "team-b",
			want:            true,
		},
		{
			name:            "same namespace is allowed even with a policy listing others",
			policy:          &RealmImportPolicy{AllowedNamespaces: []string{"team-c"}},
			sourceNamespace: "team-a",
			want:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kc := &Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "keycloak",
					Namespace: "team-a",
				},
				Spec: KeycloakSpec{
					RealmImportPolicy: tt.policy,
				},
			}

			if got := kc.AllowsImportFrom(tt.sourceNamespace); got != tt.want {
				t.Errorf("AllowsImportFrom(%q) = %v, want %v", tt.sourceNamespace, got, tt.want)
			}
		})
	}
}
