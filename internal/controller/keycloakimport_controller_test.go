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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	ssov1alpha1 "github.com/stakater/rhbk-operator/api/v1alpha1"
	"github.com/stakater/rhbk-operator/internal/resources"
)

var _ = Describe("KeycloakImport Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-realm"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "rhsso-realm",
		}
		keycloakimport := &ssov1alpha1.KeycloakImport{}
		keycloak := &ssov1alpha1.Keycloak{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "keycloak",
				Namespace: "rhsso",
			},
			Spec: ssov1alpha1.KeycloakSpec{
				Instances: &[]int32{1}[0],
			},
		}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Keycloak")
			err := k8sClient.Create(ctx, keycloak)
			Expect(err).NotTo(HaveOccurred())

			k8sClient.Create(ctx, &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "rhsso-realm",
				},
			})

			By("creating the custom resource for the Kind KeycloakImport")
			err = k8sClient.Get(ctx, typeNamespacedName, keycloakimport)
			if err != nil && errors.IsNotFound(err) {
				keycloakimport := &ssov1alpha1.KeycloakImport{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "rhsso-realm",
					},
					Spec: ssov1alpha1.KeycloakImportSpec{
						KeycloakInstance: ssov1alpha1.KeycloakInstance{
							Name:      keycloak.Name,
							Namespace: keycloak.Namespace,
						},
						JSON: `{"realm": "test-realm"}`,
					},
				}
				Expect(k8sClient.Create(ctx, keycloakimport)).To(Succeed())
			}

			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      keycloak.Name,
					Namespace: keycloak.Namespace,
				}, keycloak)
			}).Should(Succeed())

			Eventually(func() error {
				return k8sClient.Get(ctx, typeNamespacedName, keycloakimport)
			}).Should(Succeed())
		})

		AfterEach(func() {
			resource := &ssov1alpha1.KeycloakImport{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance KeycloakImport")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			By("Cleanup the custom resource for the Kind Keycloak")
			resourceKeycloak := &ssov1alpha1.Keycloak{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      keycloak.Name,
				Namespace: keycloak.Namespace,
			}, resourceKeycloak)
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Delete(ctx, resourceKeycloak)).To(Succeed())
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &KeycloakImportReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			job := &batchv1.Job{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      resources.GetImportJobName(keycloakimport),
				Namespace: keycloakimport.Namespace,
			}, job)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
