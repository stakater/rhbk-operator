# rhbk-operator vs. the official Red Hat Build of Keycloak operator

Both operators deploy the **same RHBK container image**
(`registry.redhat.io/rhbk/keycloak-rhel9`) as a StatefulSet with Infinispan
clustering, an external database, and health probes. The differences are in
the management layer around it. This doc is for anyone deciding which to run —
it is honest in both directions.

> Verified against RHBK operator docs 26.x (checked 2026-11-02) and this
> repo's code. The official operator moves fast; re-check the
> [Red Hat docs](https://docs.redhat.com/en/documentation/red_hat_build_of_keycloak/)
> before relying on a listed limitation.

## The two in one sentence each

- **Official RHBK operator** — Red Hat's supported, platform-portable,
  feature-complete operator (downstream of the upstream Keycloak operator;
  CRDs `k8s.keycloak.org/v2alpha1`).
- **rhbk-operator (this repo)** — Stakater's opinionated OpenShift layer
  (CRDs `sso.stakater.com/v1alpha1`): batteries included for TLS, Routes,
  monitoring, GitOps realm management, and per-team instances.

## Feature comparison

| Area | rhbk-operator (Stakater) | Official RHBK operator |
|---|---|---|
| Support | Community / self-supported | Red Hat supported (RHBK subscription), CVE/errata stream |
| Platform | OpenShift-only (Routes, service-CA) | Kubernetes + OpenShift |
| TLS | Automatic: service-CA serving cert + reencrypt Route | You provide `spec.http.tlsSecret` (cert-manager etc.) |
| Exposure | Route created and managed for you | Ingress/Route via `spec.ingress`, cert wiring is yours |
| Monitoring | ServiceMonitor created for you | Metrics exposed; ServiceMonitor is yours |
| Custom providers / SPIs | Downloaded into the classpath at pod start from `spec.providers` URLs | Custom optimized image build required (extensions in the CR were dropped) |
| Realm import | Realm JSON + `%.VAR%` **secret substitution**, `overrideIfExists` re-import, GitOps-friendly | `KeycloakRealmImport`: typed, **create-only** (never updates an existing realm), `placeholders` from Secrets |
| Sizing | Declarative `spec.sizing` (logins/sec, grants/sec) → resources computed from Keycloak's sizing model | Manual `spec.resources` |
| Upgrade strategies | Plain StatefulSet RollingUpdate (see caveats) | `Recreate` default (single version on the DB), `Auto` (feasibility-checked rolling), `Explicit` |
| Database | External PostgreSQL only | Multiple vendors, first-class `spec.db` (pooling, TLS) |
| NetworkPolicies | Not yet (tracked: security issues) | Supported |
| CR surface | Small + `additionalOptions` escape hatch (any `KC_*` env) | Large: hostname v2, truststores, cache config, scheduling, `unsupported.podTemplate` |
| Multi-instance / per-team | First-class direction: namespaced CR, per-CR `version`, independent upgrades | Same mechanics possible; no particular opinion |

## Why this operator exists (what the official one makes you do)

On OpenShift, the official operator still leaves you to wire TLS secrets,
create the Route/cert plumbing, add a ServiceMonitor, build a custom image
pipeline for every provider JAR, and work around create-only realm imports
(most teams end up adding Terraform or keycloak-config-cli). This operator
automates exactly that set, which is what makes **self-service, per-team
instances** practical: a team gets a working, monitored, TLS'd Keycloak from
one small CR and upgrades it by bumping `spec.version` in their GitOps repo.

## Honest trade-offs (what you give up)

1. **No vendor support for the operator layer.** The RHBK *image* is Red
   Hat's; the operator managing it is ours. Upgrade quirks, migrations and
   CVEs in the management layer are on us.
2. **No managed upgrade strategy yet.** A version bump rolls pods while old
   and new versions touch the same database concurrently — exactly what the
   official operator's `Recreate` default prevents. Until that's closed,
   treat version bumps as maintenance events.
3. **Runtime provider downloads trade immutability for convenience.**
   Startup depends on external URLs, and the same URL can serve different
   bytes tomorrow. Checksum verification is tracked in the security issues;
   the official operator's custom-image approach is the stricter model.
4. **OpenShift-only, PostgreSQL-only, alpha API.** By design — but it rules
   the operator out for vanilla-Kubernetes or non-Postgres estates.
5. **Hardening in progress.** See the `security`-labeled issues and the SEC
   epic in [roadmap.md](roadmap.md) before offering instances to untrusted
   tenants.

## When to choose which

- **Choose the official operator** when Red Hat support of the operator
  layer matters, you need safe automated version upgrades today, you're not
  on OpenShift, or you need a non-Postgres database.
- **Choose this operator** when you run OpenShift, want per-team instances
  with GitOps realm management (secret substitution, re-import), and accept
  owning the management layer.
- **Middle path:** the official operator plus a thin Helm/Argo CD layer for
  the Route, certs, and ServiceMonitor — more moving parts, but vendor
  support is retained.

## Sources

- [RHBK Operator Guide — realm import](https://docs.redhat.com/en/documentation/red_hat_build_of_keycloak/26.4/html/operator_guide/realm-import-) (create-only, placeholders)
- [RHBK Migration Guide — operator](https://docs.redhat.com/en/documentation/red_hat_build_of_keycloak/26.4/html/migration_guide/migrating-operator) (extensions dropped, custom images)
- [Keycloak — avoiding downtime with rolling updates](https://keycloak.org/operator/rolling-updates) (update strategies)
- [Keycloak — CPU/memory sizing](https://www.keycloak.org/high-availability/concepts-memory-and-cpu-sizing) (the model behind `spec.sizing`)
