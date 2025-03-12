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

package controller

import (
	"context"

	"github.com/stakater/rhbk-operator/internal/resources/realm"
	"github.com/stakater/rhbk-operator/internal/resources/rhbk"

	"github.com/redhat-cop/operator-utils/pkg/util/apis"
	"github.com/stakater/rhbk-operator/internal/constants"
	"github.com/stakater/rhbk-operator/internal/resources"
	"github.com/stakater/rhbk-operator/test/utils/yaml"
	v13 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ssov1alpha1 "github.com/stakater/rhbk-operator/api/v1alpha1"
)

var _ = Describe("KeycloakImport Controller", func() {
	Context("When reconciling a resource", func() {
		ctx := context.Background()
		var keycloak *ssov1alpha1.Keycloak
		var keycloakImport *ssov1alpha1.KeycloakImport
		var realmSecret *v1.Secret

		BeforeEach(func() {
			keycloak = &ssov1alpha1.Keycloak{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "keycloak",
					Namespace: "rhbk-instance",
				},
			}

			By("creating the custom resource for the Kind Keycloak")
			err := k8sClient.Get(ctx, kclient.ObjectKeyFromObject(keycloak), keycloak)
			if err != nil && errors.IsNotFound(err) {
				yaml.GetResourceFromFile("keycloak.yaml", keycloak)
				Expect(k8sClient.Create(ctx, keycloak)).To(Succeed())
			}

			keycloakImport = &ssov1alpha1.KeycloakImport{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "realm-import",
					Namespace: "rhbk-import",
				},
				Spec: ssov1alpha1.KeycloakImportSpec{
					KeycloakInstance: ssov1alpha1.KeycloakInstance{
						Name:      keycloak.Name,
						Namespace: keycloak.Namespace,
					},
				},
			}

			realmSecret = &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "realm-secret",
					Namespace: "rhbk-import",
				},
			}

			By("creating the realm import secret")
			yaml.GetResourceFromFile("realm-secret.yaml", realmSecret)
			Expect(k8sClient.Create(ctx, realmSecret)).To(Succeed())

			By("creating the custom resource for the Kind KeycloakImport")
			yaml.GetResourceFromFile("keycloak-import.yaml", keycloakImport)
			Expect(k8sClient.Create(ctx, keycloakImport)).To(Succeed())
		})

		AfterEach(func() {
			By("Cleanup resource Keycloak")
			DeleteIfExist(ctx, keycloak)

			By("Cleanup resource realm secrets")
			DeleteIfExist(ctx, realmSecret)

			By("Cleanup resource KeycloakImport")
			DeleteIfExist(ctx, keycloakImport)

			By("Cleanup import secret")
			secret := GetImportSecret(ctx, keycloakImport)
			if secret != nil {
				DeleteIfExist(ctx, secret)
			}

			By("Cleanup import job")
			job := GetImportJob(ctx, keycloakImport)
			if job != nil {
				DeleteIfExist(ctx, job)
			}
		})

		It("should wait for keycloak to be ready", func() {
			SetKeycloakReady(ctx, kclient.ObjectKeyFromObject(keycloak), metav1.ConditionFalse)
			ReconcileKeycloakImport(ctx, keycloakImport)
			Expect(keycloakImport.Status.IsReady()).To(BeFalse())
			Expect(keycloakImport.Status.Conditions.ConditionMsg(apis.ReconcileSuccess)).To(Equal("RHBK instance not ready"))

			SetKeycloakReady(ctx, kclient.ObjectKeyFromObject(keycloak), metav1.ConditionTrue)
			ReconcileKeycloakImport(ctx, keycloakImport)
			Expect(keycloakImport.Status.IsReady()).To(BeFalse())
			Expect(keycloakImport.Status.Conditions.ConditionMsg(apis.ReconcileSuccess)).To(Equal("RHBK deployment not ready. statefulsets.apps \"keycloak\" not found"))
		})

		It("should create import secret", func() {
			kcKey := kclient.ObjectKeyFromObject(keycloak)
			ReconcileKeycloak(ctx, kcKey)
			FakeStatefulSetReady(ctx, kclient.ObjectKey{
				Name:      rhbk.GetStatefulSetName(keycloak),
				Namespace: keycloak.Namespace,
			})
			SetKeycloakReady(ctx, kclient.ObjectKeyFromObject(keycloak), metav1.ConditionTrue)
			ReconcileKeycloakImport(ctx, keycloakImport)

			secret := GetImportSecret(ctx, keycloakImport)
			Expect(secret.Labels).To(HaveKeyWithValue(constants.RHBKImportOwnerLabel, keycloakImport.Name))
			Expect(secret.Labels).To(HaveKeyWithValue(constants.RHBKImportNamespaceLabel, keycloakImport.Namespace))
		})

		It("should sync import job", func() {
			kcKey := kclient.ObjectKeyFromObject(keycloak)
			ReconcileKeycloak(ctx, kcKey)
			FakeStatefulSetReady(ctx, kclient.ObjectKey{
				Name:      rhbk.GetStatefulSetName(keycloak),
				Namespace: keycloak.Namespace,
			})
			SetKeycloakReady(ctx, kclient.ObjectKeyFromObject(keycloak), metav1.ConditionTrue)
			ReconcileKeycloakImport(ctx, keycloakImport)

			job := GetImportJob(ctx, keycloakImport)
			Expect(job.Labels).To(HaveKeyWithValue(constants.RHBKImportOwnerLabel, keycloakImport.Name))
			Expect(job.Labels).To(HaveKeyWithValue(constants.RHBKImportNamespaceLabel, keycloakImport.Namespace))

			Expect(keycloakImport.Status.ConditionMsg(apis.ReconcileSuccess)).To(Equal("Wait for new import job to be ready"))

			SetJobCompleteStatus(ctx, keycloakImport, v1.ConditionTrue)
			ReconcileKeycloakImport(ctx, keycloakImport)
			Expect(GetKeycloakStatefulSet(ctx, keycloak).Spec.Template.Annotations).To(HaveKeyWithValue(realm.GetImportJobAnnotation(keycloakImport), GetImportSecret(ctx, keycloakImport).ResourceVersion))

			ReconcileKeycloakImport(ctx, keycloakImport)
			Expect(keycloakImport.Status.IsReady()).To(BeTrue())
		})
	})
})

func GetImportSecret(ctx context.Context, kci *ssov1alpha1.KeycloakImport) *v1.Secret {
	secret := &v1.Secret{}
	err := k8sClient.Get(ctx, kclient.ObjectKey{
		Name:      realm.GetImportJobSecretName(kci),
		Namespace: kci.Spec.KeycloakInstance.Namespace,
	}, secret)

	if kclient.IgnoreNotFound(err) != nil || errors.IsNotFound(err) {
		return nil
	}

	return secret
}

func GetImportJob(ctx context.Context, kci *ssov1alpha1.KeycloakImport) *v12.Job {
	job := &v12.Job{}
	err := k8sClient.Get(ctx, kclient.ObjectKey{
		Name:      realm.GetImportJobName(kci),
		Namespace: kci.Spec.KeycloakInstance.Namespace,
	}, job)

	if kclient.IgnoreNotFound(err) != nil || errors.IsNotFound(err) {
		return nil
	}

	return job
}

func SetKeycloakReady(ctx context.Context, key kclient.ObjectKey, status metav1.ConditionStatus) {
	kc := &ssov1alpha1.Keycloak{}
	err := k8sClient.Get(ctx, key, kc)
	Expect(err).NotTo(HaveOccurred())

	kc.Status.Conditions.SetReady(status)
	err = k8sClient.Status().Update(ctx, kc)
	Expect(err).NotTo(HaveOccurred())
}

func FakeStatefulSetReady(ctx context.Context, key kclient.ObjectKey) {
	sts := &v13.StatefulSet{}
	err := k8sClient.Get(ctx, key, sts)
	Expect(err).NotTo(HaveOccurred())

	sts.Status.Replicas = *sts.Spec.Replicas
	sts.Status.ReadyReplicas = *sts.Spec.Replicas
	err = k8sClient.Status().Update(ctx, sts)
	Expect(err).NotTo(HaveOccurred())
}

func FakeKeycloakReady(ctx context.Context, key kclient.ObjectKey) {
	kc := &ssov1alpha1.Keycloak{}
	err := k8sClient.Get(ctx, key, kc)
	Expect(err).NotTo(HaveOccurred())

	kc.Status.SetReady(metav1.ConditionTrue)
	err = k8sClient.Status().Update(ctx, kc)
	Expect(err).NotTo(HaveOccurred())
}

func SetJobCompleteStatus(ctx context.Context, kci *ssov1alpha1.KeycloakImport, status v1.ConditionStatus) {
	job := GetImportJob(ctx, kci)
	job.Status = v12.JobStatus{
		Conditions: []v12.JobCondition{
			{Type: v12.JobComplete, Status: status},
		},
	}

	Expect(k8sClient.Status().Update(ctx, job)).NotTo(HaveOccurred())
	job.Labels[realm.GetImportJobAnnotation(kci)] = GetImportSecret(ctx, kci).ResourceVersion
	Expect(k8sClient.Update(ctx, job)).NotTo(HaveOccurred())
	Expect(resources.IsJobCompleted(job)).To(BeTrue())
}

func ReconcileKeycloakImport(ctx context.Context, kc *ssov1alpha1.KeycloakImport) {
	controllerReconciler := &KeycloakImportReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
	}

	_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
		NamespacedName: kclient.ObjectKeyFromObject(kc),
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Get(ctx, kclient.ObjectKeyFromObject(kc), kc)).ToNot(HaveOccurred())
}
