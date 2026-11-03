package utils

import (
	appsv1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreatePostgresDB deploys an ephemeral PostgreSQL (plain apps/v1 Deployment +
// Service) for e2e tests and waits until it is Available. The previous
// implementation used the postgresql-ephemeral template (DeploymentConfig),
// which is deprecated since OpenShift 4.14 and absent on current clusters.
func CreatePostgresDB(name string, dbUser string, password string, ns string) {
	ApplyFixtureTemplate("./test/e2e/fixtures/postgreSQL/postgres.yaml", ns, map[string]interface{}{
		"name":       name,
		"dbUser":     dbUser,
		"dbPassword": password,
	})

	deployment := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}

	WaitForResource(deployment, func() bool {
		for _, condition := range deployment.Status.Conditions {
			if condition.Type == appsv1.DeploymentAvailable && condition.Status == v12.ConditionTrue {
				return true
			}
		}
		return false
	}, "2m", "1s")
}
