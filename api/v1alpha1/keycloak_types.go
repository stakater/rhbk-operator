/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KeycloakSpec defines the desired state of Keycloak
type KeycloakSpec struct {
	// +required
	// PostgreSQL Database configurations
	Database *PostgresDatabase `json:"database"`

	// +optional
	// Extra options to load, or override existing ENVs
	AdditionalOptions []SecretOptionVar `json:"additionalOptions,omitempty"`

	// +required
	// Number of instances
	Instances *int32 `json:"instances"`

	// +optional
	// Trusted CA bundle from configmap
	TrustedCABundles *v1.LocalObjectReference `json:"trustedCABundles,omitempty"`

	// +optional
	// Admin credentials
	Admin AdminUser `json:"admin,omitempty"`

	// +optional
	// Custom providers & SPIs to add to the RHBK installation
	Providers []Provider `json:"providers,omitempty"`

	// Configurations for hostname related options
	// Default proxy is false
	NetworkConfig *NetworkConfig `json:"networkOptions,omitempty"`

	// +optional
	// Realm sizing
	Sizing *RealmSizing `json:"sizing,omitempty"`
}

type NetworkConfig struct {
	// Enable proxy mode will set
	// KC_PROXY_HEADERS=xforwarded
	// PROXY is expected to block paths according to https://www.keycloak.org/server/reverseproxy
	Proxy bool `json:"proxy,omitempty"`

	// Required if Proxy is disabled
	Hostname string `json:"hostname,omitempty"`
}

type Provider struct {
	Name string       `json:"name"`
	URL  SecretOption `json:"url"`
}

type AdminUser struct {
	Username SecretOption `json:"username,omitempty"`
	Password SecretOption `json:"password,omitempty"`
}

type Features struct {
	Enabled  []string `json:"enabled,omitempty"`
	Disabled []string `json:"disabled,omitempty"`
}

type PostgresDatabase struct {
	Host     SecretOption `json:"host"`
	Port     SecretOption `json:"port"`
	User     SecretOption `json:"user"`
	Password SecretOption `json:"password"`
}

type SecretOptionVar struct {
	Name   string                `json:"name"`
	Value  string                `json:"value,omitempty"`
	Secret *v1.SecretKeySelector `json:"secret,omitempty"`
}

type SecretOption struct {
	Value  string                `json:"value,omitempty"`
	Secret *v1.SecretKeySelector `json:"secret,omitempty"`
}

// KeycloakStatus defines the observed state of Keycloak
type KeycloakStatus struct {
	Conditions `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Keycloak is the Schema for the keycloaks API
type Keycloak struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakSpec   `json:"spec,omitempty"`
	Status KeycloakStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KeycloakList contains a list of Keycloak
type KeycloakList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Keycloak `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Keycloak{}, &KeycloakList{})
}
