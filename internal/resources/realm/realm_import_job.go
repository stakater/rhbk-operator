package realm

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/stakater/rhbk-operator/internal/resources"

	"github.com/stakater/rhbk-operator/api/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/batch/v1"
	v14 "k8s.io/api/core/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetImportJobName(cr *v1alpha1.KeycloakImport) string {
	return fmt.Sprintf("%s-import", cr.Name)
}

func GetImportJobSecretVolumeName(cr *v1alpha1.KeycloakImport) string {
	return GetImportJobSecretName(cr) + "-volume"
}

func GetRealmMountPath(cr *v1alpha1.KeycloakImport) string {
	return fmt.Sprintf("/mnt/realm-import/%s-realm.json", cr.Name)
}

func GetImportJobAnnotation(cr *v1alpha1.KeycloakImport) string {
	return fmt.Sprintf("%s/%s", cr.Name, cr.Namespace)
}

type RealmImportJob struct {
	KeycloakImport *v1alpha1.KeycloakImport
	StatefulSet    *v1.StatefulSet
}

func Build(cr *v1alpha1.KeycloakImport, sts *v1.StatefulSet, revision string) (*v12.Job, error) {
	ownerLabels := resources.GetOwnerLabels(cr.Name, cr.Namespace)
	ownerLabels[GetImportJobAnnotation(cr)] = revision
	resources.DecorateDefaultLabels(ownerLabels)

	template := sts.Spec.Template.DeepCopy()
	template.Labels = ownerLabels
	kcContainer := &template.Spec.Containers[0]

	toModify := map[string]string{
		"KC_CACHE":          "local",
		"KC_HEALTH_ENABLED": "false",
		"KC_CACHE_STACK":    "",
	}

	// Setup ENVs for a job
	var next []v14.EnvVar
	for _, v := range kcContainer.Env {
		if n, ok := toModify[v.Name]; ok {
			if n == "" {
				continue
			}

			next = append(next, v14.EnvVar{
				Name:  v.Name,
				Value: n,
			})
		} else {
			next = append(next, v)
		}
	}

	// Setup volume for mounting realm JSON
	template.Spec.Volumes = append(template.Spec.Volumes, v14.Volume{
		Name: GetImportJobSecretVolumeName(cr),
		VolumeSource: v14.VolumeSource{
			Secret: &v14.SecretVolumeSource{
				SecretName: GetImportJobSecretName(cr),
			},
		},
	})

	kcContainer.VolumeMounts = append(kcContainer.VolumeMounts, v14.VolumeMount{
		Name:      GetImportJobSecretVolumeName(cr),
		ReadOnly:  true,
		MountPath: "/mnt/realm-import",
	})

	kcContainer.Env = next

	// Remove probes
	kcContainer.ReadinessProbe = nil
	kcContainer.LivenessProbe = nil
	kcContainer.StartupProbe = nil

	cmd := []string{
		"/bin/bash",
	}

	buildProviders := "/opt/keycloak/bin/kc.sh --verbose build && "
	args := []string{
		"-c",
		fmt.Sprintf(`%s/opt/keycloak/bin/kc.sh --verbose import --optimized --file='%s' --override=%t`,
			buildProviders,
			GetRealmMountPath(cr),
			cr.Spec.OverrideIfExists),
	}

	kcContainer.Command = cmd
	kcContainer.Args = args

	template.Spec.RestartPolicy = v14.RestartPolicyNever
	job := &v12.Job{
		ObjectMeta: v13.ObjectMeta{
			Name:      GetImportJobName(cr),
			Namespace: sts.Namespace,
			Labels:    ownerLabels,
		},
		Spec: v12.JobSpec{
			Template:     *template,
			BackoffLimit: &[]int32{1}[0],
		},
	}

	return job, nil
}

func GetImportJobs(ctx context.Context, kc client.Client, kci *v1alpha1.KeycloakImport) (*v12.JobList, error) {
	kcNamespace := kci.Spec.KeycloakInstance.Namespace
	ownerLabels := labels.SelectorFromSet(resources.GetOwnerLabels(kci.Name, kci.Namespace))

	jobs := &v12.JobList{}
	err := kc.List(ctx, jobs, client.InNamespace(kcNamespace), client.MatchingLabelsSelector{
		Selector: ownerLabels,
	})

	if err != nil {
		return nil, err
	}

	return jobs, nil
}
