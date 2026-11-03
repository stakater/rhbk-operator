# rhbk-operator

Run **Red Hat Build of Keycloak** on OpenShift with two CRs and zero TLS/Route/monitoring toil.

Stakater's opinionated operator for RHBK: it deploys the official
`registry.redhat.io/rhbk/keycloak-rhel9` image and automates everything the
[official Red Hat operator leaves to you](docs/comparison-with-official-operator.md)
on OpenShift — so each team can run **its own instance** and upgrade on its own
schedule.

> **Status:** alpha (`v1alpha1`, v0.0.x). OpenShift-only by design.

## What you get

| You declare | The operator handles |
|---|---|
| `instances`, `version` | StatefulSet, Infinispan clustering (JGroups DNS ping), probes, rolling updates |
| nothing | **TLS** via OpenShift service-CA, reencrypt **Route**, **ServiceMonitor** |
| `database` (secret refs) | PostgreSQL wiring, connection pool |
| `KeycloakImport` CR | Realm import job with **secret substitution** into the realm JSON |
| `providers` | Custom SPI JARs fetched into the Keycloak classpath |
| `sizing` (logins/sec, grants/sec) | CPU/memory computed from [Keycloak's sizing model](https://www.keycloak.org/high-availability/concepts-memory-and-cpu-sizing) |

## Prerequisites

- OpenShift cluster (Routes, service-CA)
- External **PostgreSQL** (one per instance)
- Pull access to `registry.redhat.io`

## Quickstart

```yaml
apiVersion: sso.stakater.com/v1alpha1
kind: Keycloak
metadata:
  name: keycloak
  namespace: team-a
spec:
  instances: 2
  version: "26.0-6"
  admin:
    username: { secret: { name: admin, key: username } }
    password: { secret: { name: admin, key: password } }
  database:
    host:     { secret: { name: rhbk-pguser, key: host } }
    port:     { secret: { name: rhbk-pguser, key: port } }
    user:     { secret: { name: rhbk-pguser, key: user } }
    password: { secret: { name: rhbk-pguser, key: password } }
```

Wait for `status.conditions` to report Ready, then log in at the created Route.
Always reference credentials from Secrets — never inline values.

### Import a realm

```yaml
apiVersion: sso.stakater.com/v1alpha1
kind: KeycloakImport
metadata:
  name: team-a-realm
  namespace: team-a
spec:
  keycloakInstance: { name: keycloak, namespace: team-a }
  overrideIfExists: true
  substitutions:
    - name: GOOGLE_CLIENT_SECRET
      secret: { name: idp-secrets, key: google }
  json: |
    {
      "realm": "team-a",
      "identityProviders": [{ "config": { "clientSecret": "%.GOOGLE_CLIENT_SECRET%" } }]
    }
```

`%.VAR%` placeholders are resolved from Secrets at import time, so the realm
JSON stays safe to commit to Git.

## CRDs

| CRD | Purpose | Key fields |
|---|---|---|
| `Keycloak` | One RHBK deployment | `version`, `instances`, `database`, `admin`, `providers`, `networkOptions`, `sizing`, `trustedCABundles`, `additionalOptions` (escape hatch: any `KC_*` env) |
| `KeycloakImport` | Declarative realm import | `keycloakInstance`, `json`, `substitutions`, `overrideIfExists` |

Full samples in [config/samples/](config/samples/).

## Per-team instances

The operator is cluster-scoped; CRs are namespaced. Give each team a namespace,
a `Keycloak` CR, and a PostgreSQL — each team bumps its own `spec.version` and
upgrades independently. Isolation hardening is in progress: see the
[roadmap](docs/roadmap.md) and the `security`-labeled issues before offering
this to untrusted tenants.

## vs. the official Red Hat operator

Short version: the official operator is supported by Red Hat and
platform-portable; this one automates the OpenShift-specific setup (certs,
Routes, monitoring), imports realms repeatably with secret substitution, and
loads providers without a custom image build. Full comparison:
[docs/comparison-with-official-operator.md](docs/comparison-with-official-operator.md).

## Developing

```sh
make test     # manifests, generate, fmt, vet, envtest suite
make lint     # golangci-lint
make install run  # CRDs + run the manager locally
```

How we work (stories, planning docs, TDD, DoD): [CLAUDE.md](CLAUDE.md) and
[docs/method.md](docs/method.md). Release flow: [DEPLOY.md](DEPLOY.md).

## License

Apache 2.0
