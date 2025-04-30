package e2e

import (
	"encoding/base64"
	"fmt"

	"github.com/Nerzal/gocloak/v12"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	deploymentConfig "github.com/openshift/api/apps/v1"
	v13 "github.com/openshift/api/route/v1"
	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stakater/rhbk-operator/api/v1alpha1"
	"github.com/stakater/rhbk-operator/test/utils"
)

var _ = Describe("RHBK", Ordered, func() {
	const rhbkNs = "rhbk-instance"
	realmNs := "e2e-realm"
	var kc *utils.Keycloak

	rhbkVars := map[string]interface{}{
		"name":        "e2e-rhbk",
		"dbUser":      "e2e",
		"dbPassword":  "test123",
		"adminSecret": "admin-secret",
		"dbHost":      "e2e-rhbk",
	}

	realmVars := map[string]interface{}{
		"name":              "simple-realm",
		"rhbkName":          rhbkVars["name"].(string),
		"rhbkNamespace":     rhbkNs,
		"replacementSecret": "simple-realm-secret",
		"secretValue":       base64.StdEncoding.EncodeToString([]byte("simple-realm")),
		"realmEnabled":      true,
	}

	BeforeAll(func() {
		By("creating namespaces")
		for _, ns := range []string{rhbkNs, realmNs} {
			_, err := utils.RunShell("oc", "new-project", ns, "||", "oc", "project", ns)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	AfterAll(func() {
		By("removing namespaces")
		for _, ns := range []string{rhbkNs, realmNs} {
			_, err := utils.Run("oc", "delete", "project", ns)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	Context("operator", func() {
		It("should create RHBK instance", func() {
			By("create postgres database")
			utils.CreatePostgresDB(
				rhbkVars["name"].(string),
				rhbkVars["dbUser"].(string),
				rhbkVars["dbPassword"].(string),
				rhbkNs,
			)
			deploymentConfig := &deploymentConfig.DeploymentConfig{
				ObjectMeta: v1.ObjectMeta{
					Name:      rhbkVars["name"].(string),
					Namespace: rhbkNs,
				},
			}

			utils.WaitForResource(deploymentConfig, func() bool {
				for _, condition := range deploymentConfig.Status.Conditions {
					if condition.Type == "Progressing" && condition.Status == v12.ConditionFalse {
						return false
					}

					if condition.Type == "Available" && condition.Status == v12.ConditionTrue {
						return true
					}
				}
				return false
			}, "2m", "1s")

			By("deploying admin secrets")
			utils.ApplyFixtureTemplate("./test/e2e/fixtures/postgreSQL/admin-secret.yaml", rhbkNs, rhbkVars)

			By("deploying rhbk instance")
			utils.ApplyFixtureTemplate("./test/e2e/fixtures/postgreSQL/rhbk.yaml", rhbkNs, rhbkVars)
			rhbk := &v1alpha1.Keycloak{
				ObjectMeta: v1.ObjectMeta{
					Name:      "e2e-rhbk",
					Namespace: rhbkNs,
				},
			}
			utils.WaitForResource(rhbk, func() bool {
				return rhbk.Status.IsReady()
			}, "5m", "1s")
			utils.MatchYAMLResource(rhbk, "rhbk", "instance")

			By("logging in to rhbk instance")
			adminSecret := &v12.Secret{
				ObjectMeta: v1.ObjectMeta{
					Name:      rhbkVars["adminSecret"].(string),
					Namespace: rhbkNs,
				},
			}
			utils.WaitForResource(adminSecret, func() bool {
				return len(adminSecret.Data) > 0
			}, "10s", "1s")

			route := &v13.Route{
				ObjectMeta: v1.ObjectMeta{
					Name:      rhbkVars["name"].(string),
					Namespace: rhbkNs,
				},
			}

			utils.WaitForResource(route, func() bool {
				return route.Spec.Host != ""
			}, "1m", "1s")
			utils.MatchYAMLResource(route, "rhbk", "url")

			var token *gocloak.JWT
			kc = utils.NewKeycloak(
				fmt.Sprintf("https://%s", route.Spec.Host),
				string(adminSecret.Data["username"]),
				string(adminSecret.Data["password"]),
			)
			Eventually(func() error {
				res, err := kc.AdminLogin("master")
				token = res
				return err
			}, "1m", "1s").ShouldNot(HaveOccurred())
			Expect(token).ToNot(BeNil())
		})

		It("should deploy import realm", func() {
			By("creating realm")
			utils.ApplyFixtureTemplate("./test/e2e/fixtures/realm-import/replacement_secret.yaml", realmNs, realmVars)
			utils.ApplyFixtureTemplate("./test/e2e/fixtures/realm-import/realm.yaml", realmNs, realmVars)

			realm := &v1alpha1.KeycloakImport{
				ObjectMeta: v1.ObjectMeta{
					Name:      realmVars["name"].(string),
					Namespace: realmNs,
				},
			}
			utils.WaitForResource(realm, func() bool {
				return realm.Status.IsReady()
			}, "1m", "1s")

			utils.MatchYAMLResource(realm, "realm", "import")

			By("checking realm")
			var realmImport *gocloak.RealmRepresentation
			Eventually(func() error {
				res, err := kc.GetRealm(realmVars["name"].(string))
				realmImport = res
				return err
			}, "3m", "1s").ShouldNot(HaveOccurred())

			decodedValue, err := base64.StdEncoding.DecodeString(realmVars["secretValue"].(string))
			Expect(err).NotTo(HaveOccurred())
			Expect(*realmImport.DisplayName).To(Equal(string(decodedValue)))
		})

		It("should update realm when secret is updated", func() {
			By("updating secrets")
			displayName := "e2e-realm-updated"
			realmVars["secretValue"] = base64.StdEncoding.EncodeToString([]byte(displayName))
			utils.ApplyFixtureTemplate("./test/e2e/fixtures/realm-import/replacement_secret.yaml", realmNs, realmVars)

			By("checking updated realm")
			Eventually(func() error {
				res, err := kc.GetRealm(realmVars["name"].(string))
				if err != nil || *res.DisplayName != displayName {
					return fmt.Errorf("Waiting for realm to be synced")
				}
				return err
			}, "3m", "1s").ShouldNot(HaveOccurred())
		})

		It("should update realm when spec is updated", func() {
			By("updating realm enabled status")
			realmVars["realmEnabled"] = false
			utils.ApplyFixtureTemplate("./test/e2e/fixtures/realm-import/realm.yaml", realmNs, realmVars)

			By("checking updated realm enabled status")
			Eventually(func() error {
				res, err := kc.GetRealm(realmVars["name"].(string))
				if err != nil || res.Enabled != nil && !*res.Enabled {
					return fmt.Errorf("Waiting for realm to be synced")
				}
				return err
			}, "3m", "1s").ShouldNot(HaveOccurred())
		})
	})
})
