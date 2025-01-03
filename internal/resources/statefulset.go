package resources

import (
	"context"
	"fmt"
	"strconv"

	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/stakater/rhbk-operator/api/v1alpha1"
)

const RHBKImage = "registry.redhat.io/rhbk/keycloak-rhel9@sha256:89f9c4680ac4be190904fefe720403f3bc43cfd245fdac23154b35e0d7a74a3b"

type RHBKStatefulSet struct {
	Keycloak *v1alpha1.Keycloak
	HostName string
	Scheme   *runtime.Scheme
	Resource *v1.StatefulSet
}

func GetStatefulSetName(cr *v1alpha1.Keycloak) string {
	return cr.Name
}

func (ks *RHBKStatefulSet) GetDBEnvs() []v12.EnvVar {
	if ks.Keycloak.Spec.Database != nil {
		return []v12.EnvVar{
			{
				Name:  "KC_DB",
				Value: "postgres",
			},
			{
				Name: "KC_DB_USERNAME",
				ValueFrom: &v12.EnvVarSource{
					SecretKeyRef: ks.Keycloak.Spec.Database.User.Secret,
				},
			},
			{
				Name: "KC_DB_PASSWORD",
				ValueFrom: &v12.EnvVarSource{
					SecretKeyRef: ks.Keycloak.Spec.Database.Password.Secret,
				},
			},
			{
				Name: "KC_DB_URL_HOST",
				ValueFrom: &v12.EnvVarSource{
					SecretKeyRef: ks.Keycloak.Spec.Database.Host.Secret,
				},
			},
			{
				Name: "KC_DB_URL_PORT",
				ValueFrom: &v12.EnvVarSource{
					SecretKeyRef: ks.Keycloak.Spec.Database.Port.Secret,
				},
			},
			{
				Name:  "KC_DB_POOL_INITIAL_SIZE",
				Value: "30",
			},
			{
				Name:  "KC_DB_POOL_MIN_SIZE",
				Value: "30",
			},
			{
				Name:  "KC_DB_POOL_MAX_SIZE",
				Value: "30",
			},
		}
	}

	return []v12.EnvVar{}
}

func (ks *RHBKStatefulSet) Build() error {
	labels := map[string]string{
		"app": "rhbk",
	}

	ks.Resource.ObjectMeta.Labels = labels
	ks.Resource.Spec = v1.StatefulSetSpec{
		Replicas: ks.Keycloak.Spec.Instances,
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		UpdateStrategy: v1.StatefulSetUpdateStrategy{
			Type: v1.RollingUpdateStatefulSetStrategyType,
			RollingUpdate: &v1.RollingUpdateStatefulSetStrategy{
				Partition: &[]int32{0}[0],
			},
		},
		RevisionHistoryLimit: &[]int32{10}[0],
		MinReadySeconds:      0,
		PersistentVolumeClaimRetentionPolicy: &v1.StatefulSetPersistentVolumeClaimRetentionPolicy{
			WhenDeleted: v1.RetainPersistentVolumeClaimRetentionPolicyType,
			WhenScaled:  v1.RetainPersistentVolumeClaimRetentionPolicyType,
		},
		Template: v12.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: v12.PodSpec{
				Containers: []v12.Container{
					{
						Name:            "rhbk",
						Image:           RHBKImage,
						ImagePullPolicy: v12.PullAlways,
						Args: []string{
							fmt.Sprintf("-Djgroups.dns.query=%s.%s", GetDiscoverySvcName(ks.Keycloak), ks.Keycloak.Namespace),
							"--verbose",
							"start",
						},
						Ports: []v12.ContainerPort{
							{
								Name:          "https",
								ContainerPort: HttpsPort,
								Protocol:      v12.ProtocolTCP,
							},
							{
								Name:          "http",
								ContainerPort: ManagementPort,
								Protocol:      v12.ProtocolTCP,
							},
						},
						Env: append([]v12.EnvVar{
							{
								Name:  "KC_HOSTNAME",
								Value: ks.HostName,
							},
							{
								Name:  "KC_HTTP_PORT",
								Value: strconv.FormatInt(ManagementPort, 10),
							},
							{
								Name:  "KC_HTTPS_PORT",
								Value: strconv.FormatInt(HttpsPort, 10),
							},
							{
								Name:  "KC_HTTPS_CERTIFICATE_FILE",
								Value: "/mnt/certificates/tls.crt",
							},
							{
								Name:  "KC_HTTPS_CERTIFICATE_KEY_FILE",
								Value: "/mnt/certificates/tls.key",
							},
							{
								Name:  "KC_HEALTH_ENABLED",
								Value: strconv.FormatBool(true),
							},
							{
								Name:  "KC_CACHE",
								Value: "ispn",
							},
							{
								Name:  "KC_CACHE_STACK",
								Value: "kubernetes",
							},
							{
								Name: "KC_BOOTSTRAP_ADMIN_USERNAME",
								ValueFrom: &v12.EnvVarSource{
									SecretKeyRef: ks.Keycloak.Spec.Admin.Username.Secret,
								},
							},
							{
								Name: "KC_BOOTSTRAP_ADMIN_PASSWORD",
								ValueFrom: &v12.EnvVarSource{
									SecretKeyRef: ks.Keycloak.Spec.Admin.Password.Secret,
								},
							},
							{
								Name:  "KC_TRUSTSTORE_PATHS",
								Value: "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt,/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt",
							},
							{
								Name:  "KC_TRACING_SERVICE_NAME",
								Value: ks.Keycloak.Name,
							},
							{
								Name:  "KC_TRACING_RESOURCE_ATTRIBUTES",
								Value: fmt.Sprintf("k8s.namespace.name=%s", ks.Keycloak.Namespace),
							},
						}, ks.GetDBEnvs()...),
						Resources: v12.ResourceRequirements{
							Requests: v12.ResourceList{
								v12.ResourceMemory: resource.MustParse("1700Mi"),
							},
							Limits: v12.ResourceList{
								v12.ResourceMemory: resource.MustParse("2Gi"),
							},
						},
						LivenessProbe: &v12.Probe{
							ProbeHandler: v12.ProbeHandler{
								HTTPGet: &v12.HTTPGetAction{
									Path: "/health/live",
									Port: intstr.IntOrString{
										IntVal: ManagementPort,
									},
									Scheme: v12.URISchemeHTTPS,
								},
							},
							PeriodSeconds:    10,
							FailureThreshold: 3,
						},
						ReadinessProbe: &v12.Probe{
							ProbeHandler: v12.ProbeHandler{
								HTTPGet: &v12.HTTPGetAction{
									Path: "/health/ready",
									Port: intstr.IntOrString{
										IntVal: ManagementPort,
									},
									Scheme: v12.URISchemeHTTPS,
								},
							},
							PeriodSeconds:    10,
							FailureThreshold: 3,
						},
						StartupProbe: &v12.Probe{
							ProbeHandler: v12.ProbeHandler{
								HTTPGet: &v12.HTTPGetAction{
									Path: "/health/started",
									Port: intstr.IntOrString{
										IntVal: ManagementPort,
									},
									Scheme: v12.URISchemeHTTPS,
								},
							},
							PeriodSeconds:    1,
							FailureThreshold: 600,
						},
						VolumeMounts: []v12.VolumeMount{
							{
								Name:      "keycloak-tls-certificates",
								MountPath: "/mnt/certificates",
							},
							{
								Name:      "providers",
								MountPath: ProvidersPATH,
							},
						},
					},
				},
				Volumes: []v12.Volume{
					{
						Name: "keycloak-tls-certificates",
						VolumeSource: v12.VolumeSource{
							Secret: &v12.SecretVolumeSource{
								SecretName:  GetTLSSecretName(ks.Keycloak),
								DefaultMode: &[]int32{420}[0],
								Optional:    &[]bool{false}[0],
							},
						},
					},
					{
						Name: "providers",
						VolumeSource: v12.VolumeSource{
							EmptyDir: &v12.EmptyDirVolumeSource{},
						},
					},
				},
			},
		},
		VolumeClaimTemplates: []v12.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "providers",
				},
				Spec: v12.PersistentVolumeClaimSpec{
					AccessModes: []v12.PersistentVolumeAccessMode{v12.ReadWriteOnce},
					Resources: v12.VolumeResourceRequirements{
						Requests: v12.ResourceList{
							v12.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			},
		},
	}

	err := controllerutil.SetControllerReference(ks.Keycloak, ks.Resource, ks.Scheme)
	if err != nil {
		return err
	}

	return nil
}

func (ks *RHBKStatefulSet) CreateOrUpdate(ctx context.Context, c client.Client) error {
	ks.Resource = &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetStatefulSetName(ks.Keycloak),
			Namespace: ks.Keycloak.Namespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, c, ks.Resource, ks.Build)

	return err
}
