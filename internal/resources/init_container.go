package resources

import (
	"fmt"
	"github.com/stakater/rhbk-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"regexp"
	"strings"
	"unicode"
)

const BusyboxImage = "registry.access.redhat.com/ubi8/ubi:8.10-1088"
const ProvidersPATH = "/opt/keycloak/providers"

func GetInitContainer(cr *v1alpha1.KeycloakImport) []v1.Container {
	if len(cr.Spec.Providers) == 0 {
		return nil
	}

	runArg := fmt.Sprintf("mkdir -p %s; curl -LJ --show-error --capath /var/run/secrets/kubernetes.io", ProvidersPATH)
	downloadContainer := v1.Container{
		Name:  "fetch",
		Image: BusyboxImage,
		Env:   []v1.EnvVar{},
		Command: []string{
			"/bin/bash",
		},
		Args: []string{
			"-c",
		},
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      "providers",
				MountPath: ProvidersPATH,
			},
		},
	}

	for _, p := range cr.Spec.Providers {
		envName := ConvertToEnvName(p.Name)
		if p.URL.Value != "" {
			downloadContainer.Env = append(downloadContainer.Env, v1.EnvVar{
				Name:  envName,
				Value: p.URL.Value,
			})
		} else {
			downloadContainer.Env = append(downloadContainer.Env, v1.EnvVar{
				Name: envName,
				ValueFrom: &v1.EnvVarSource{
					SecretKeyRef: p.URL.Secret,
				},
			})
		}

		runArg += fmt.Sprintf(" -o %s/%s $(%s)", ProvidersPATH, p.Name, envName)
	}

	downloadContainer.Args = []string{
		"-c",
		runArg,
	}
	return []v1.Container{
		downloadContainer,
	}
}

func ConvertToEnvName(input string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	sanitized := re.ReplaceAllString(input, "_")

	if unicode.IsDigit(rune(sanitized[0])) {
		sanitized = "_" + sanitized
	}

	return strings.ToUpper(sanitized)
}
