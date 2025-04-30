package utils

import "fmt"

func CreatePostgresDB(name string, dbUser string, password string, ns string) (string, error) {
	return RunShell(fmt.Sprintf(
		"oc new-app --template=postgresql-ephemeral "+
			"-p POSTGRESQL_USER=%s "+
			"-p POSTGRESQL_PASSWORD=%s "+
			"-p POSTGRESQL_DATABASE=keycloak "+
			"-p DATABASE_SERVICE_NAME=%s "+
			"-p POSTGRESQL_VERSION=15-el8 "+
			"-n %s",
		dbUser, password, name, ns))
}
