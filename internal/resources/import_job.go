package resources

import (
	"fmt"
	"github.com/stakater/rhbk-operator/api/v1alpha1"
	"github.com/stakater/rhbk-operator/internal/constants"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/batch/v1"
	v14 "k8s.io/api/core/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type ImportJob struct {
	ImportCR    *v1alpha1.KeycloakImport
	Scheme      *runtime.Scheme
	StatefulSet *v1.StatefulSet
	Job         *v12.Job
}

func GetImportJobName(cr *v1alpha1.KeycloakImport) string {
	return cr.Name
}

func GetImportJobSecretVolumeName(cr *v1alpha1.KeycloakImport) string {
	return GetImportJobSecretName(cr) + "-volume"
}

func GetRealmMountPath(cr *v1alpha1.KeycloakImport) string {
	return fmt.Sprintf("/mnt/realm-import/%s-realm.json", cr.Name)
}

func (j *ImportJob) Build() error {

	template := j.StatefulSet.Spec.Template.DeepCopy()
	template.Labels["app"] = "realm-import-job"
	kcContainer := &template.Spec.Containers[0]

	envs := map[string]*v1alpha1.SecretOption{
		"KC_CACHE": {
			Value: "local",
		},
		"KC_HEALTH_ENABLED": {
			Value: "false",
		},
		"KC_CACHE_STACK": nil,
	}

	// Setup ENV replacement
	for _, substitution := range j.ImportCR.Spec.Substitutions {
		envs[substitution.Name] = &v1alpha1.SecretOption{
			Value:  substitution.Value,
			Secret: substitution.Secret,
		}
	}

	// Setup ENVs for a job
	var next []v14.EnvVar
	for _, v := range kcContainer.Env {
		if n, ok := envs[v.Name]; ok {
			if n == nil {
				continue
			}

			nextVar := v14.EnvVar{
				Name:  v.Name,
				Value: n.Value,
			}

			if n.Secret != nil {
				nextVar.ValueFrom = &v14.EnvVarSource{
					SecretKeyRef: n.Secret,
				}
			}

			next = append(next, nextVar)
		} else {
			next = append(next, v)
		}
	}

	// Setup volume for mounting realm JSON
	template.Spec.Volumes = append(template.Spec.Volumes, v14.Volume{
		Name: GetImportJobSecretVolumeName(j.ImportCR),
		VolumeSource: v14.VolumeSource{
			Secret: &v14.SecretVolumeSource{
				SecretName: GetImportJobSecretName(j.ImportCR),
			},
		},
	})

	kcContainer.VolumeMounts = append(kcContainer.VolumeMounts, v14.VolumeMount{
		Name:      GetImportJobSecretVolumeName(j.ImportCR),
		ReadOnly:  true,
		MountPath: "/mnt/realm-import",
	})

	// Remove probes
	kcContainer.ReadinessProbe = nil
	kcContainer.LivenessProbe = nil
	kcContainer.StartupProbe = nil

	cmd := []string{
		"/bin/bash",
	}

	args := []string{
		"-c",
		fmt.Sprintf(`/opt/keycloak/bin/kc.sh --verbose build && /opt/keycloak/bin/kc.sh --verbose import --optimized --file='%s' --override=%t`,
			GetRealmMountPath(j.ImportCR),
			j.ImportCR.Spec.OverrideIfExists),
	}

	kcContainer.Command = cmd
	kcContainer.Args = args

	template.Spec.RestartPolicy = v14.RestartPolicyNever
	j.Job = &v12.Job{
		ObjectMeta: v13.ObjectMeta{
			Name:      GetImportJobName(j.ImportCR),
			Namespace: j.StatefulSet.Namespace,
			Labels: map[string]string{
				constants.RHBKRealmImportLabel: j.ImportCR.Name,
			},
		},
		Spec: v12.JobSpec{
			Template:     *template,
			BackoffLimit: &[]int32{1}[0],
		},
	}

	err := controllerutil.SetControllerReference(j.ImportCR, j.Job, j.Scheme)
	if err != nil {
		return err
	}

	return nil
}
