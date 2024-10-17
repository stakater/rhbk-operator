package resources

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/stakater/rhbk-operator/api/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
)

const IMAGE = "registry.redhat.io/rhbk/keycloak-rhel9:24-17"
const DISCOVERY_SERVICE_NAME = ""
const ANNOTATION_HASH = "rhbk.stakater.com/hash"

type RHBKStatefulSet struct {
	Keycloak *v1alpha1.Keycloak
	HostName string
	Scheme   *runtime.Scheme
	Resource *v1.StatefulSet
}

func GetStatefulSetName(cr *v1alpha1.Keycloak) string {
	return cr.Name
}

func (ks *RHBKStatefulSet) Build() error {
	labels := map[string]string{
		"app": "rhbk",
	}

	ks.Resource = &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetStatefulSetName(ks.Keycloak),
			Namespace: ks.Keycloak.Namespace,
			Labels:    labels,
			Annotations: map[string]string{
				ANNOTATION_HASH: computeHash(ks.Keycloak),
			},
		},
		Spec: v1.StatefulSetSpec{
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
							Image:           IMAGE,
							ImagePullPolicy: v12.PullAlways,
							Args: []string{
								fmt.Sprintf("-Djgroups.dns.query=%s.%s", DISCOVERY_SERVICE_NAME, ks.Keycloak.Namespace),
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
									ContainerPort: HttpPort,
									Protocol:      v12.ProtocolTCP,
								},
							},
							Env: []v12.EnvVar{
								{
									Name:  "KC_HOSTNAME",
									Value: ks.HostName,
								},
								{
									Name:  "KC_HTTP_ENABLED",
									Value: strconv.FormatBool(true),
								},
								{
									Name:  "KC_HTTP_PORT",
									Value: strconv.FormatInt(HttpPort, 10),
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
									Name:  "KC_PROXY",
									Value: "passthrough",
								},
								{
									Name: "KEYCLOAK_ADMIN",
									ValueFrom: &v12.EnvVarSource{
										SecretKeyRef: ks.Keycloak.Spec.Admin.Username.Secret,
									},
								},
								{
									Name: "KEYCLOAK_ADMIN_PASSWORD",
									ValueFrom: &v12.EnvVarSource{
										SecretKeyRef: ks.Keycloak.Spec.Admin.Password.Secret,
									},
								},
								{
									Name:  "KC_TRUSTSTORE_PATHS",
									Value: "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt,/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt",
								},
							},
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
											IntVal: HttpsPort,
										},
										Scheme: v12.URISchemeHTTPS,
									},
								},
							},
							ReadinessProbe: &v12.Probe{
								ProbeHandler: v12.ProbeHandler{
									HTTPGet: &v12.HTTPGetAction{
										Path: "/health/ready",
										Port: intstr.IntOrString{
											IntVal: HttpsPort,
										},
										Scheme: v12.URISchemeHTTPS,
									},
								},
							},
							StartupProbe: &v12.Probe{
								ProbeHandler: v12.ProbeHandler{
									HTTPGet: &v12.HTTPGetAction{
										Path: "/health/started",
										Port: intstr.IntOrString{
											IntVal: HttpsPort,
										},
										Scheme: v12.URISchemeHTTPS,
									},
								},
							},
							VolumeMounts: []v12.VolumeMount{
								{
									Name:      "keycloak-tls-certificates",
									MountPath: "/mnt/certificates",
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
					},
				},
			},
		},
	}

	err := controllerutil.SetOwnerReference(ks.Keycloak, ks.Resource, ks.Scheme)
	if err != nil {
		return err
	}

	return nil
}

func (ks *RHBKStatefulSet) CreateOrUpdate(ctx context.Context, c client.Client) error {
	err := ks.Build()

	if err != nil {
		return err
	}

	_, err = controllerutil.CreateOrUpdate(ctx, c, ks.Resource, func() error { return nil })
	return err
}

func computeHash(cr *v1alpha1.Keycloak) string {
	specBytes, err := json.Marshal(cr.Spec)

	if err != nil {
		return ""
	}

	hash := sha256.Sum256(specBytes)
	return fmt.Sprintf("%x", hash[:])
}
