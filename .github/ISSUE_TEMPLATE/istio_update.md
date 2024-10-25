---
name: Istio Update
about: Create an Istio update issue
title: "Istio {version} update"
labels: kind/feature
assignees: ''

---
**Description**

Update the Kyma Istio module to use a new Istio version.
Update the Istio version and dependencies, adjust tests and documentation if needed.
The upgrade needs to have zero downtime for production-ready settings.

ACs:
- [ ] Review Istio RNs.
- [ ] Verify that the new Istio version doesn't introduce features that transition to a new [phase](https://istio.io/latest/docs/releases/feature-stages/) in Istio, potentially affecting Kyma's Istio behavior. If such changes are identified, discuss them with the team to determine the best course of action.
- [ ] New Istio minor should be bumped on the `main` / New Istio patch should be bumped on the `main` and the latest release branch.
- [ ] Images of the new Istio version are synced to the Kyma registry by adding them to the [external-images.yaml](https://github.com/kyma-project/istio/blob/main/external-images.yaml).
- [ ] Prepare Kyma runtime Istio RNs based on open-source Istio RNs.
- [ ] Istio installs and upgrades to a new version.
- [ ] Tests and documentation updated if needed.
- [ ] Verify that sidecars are in sync with Control Plane.
- [ ] Istio and Envoy Version updated in the [`README.md`](https://github.com/kyma-project/istio) and [`/docs/user/README.md`](https://github.com/kyma-project/istio) files. You can use the scripts `scripts/get_module_istio_version.sh` and `scripts/get_module_envoy_version.sh` to extract the versions.
- [ ] Check **compatibilityVersion** of the previous minor version. You can find it in the [`helm-profiles`](https://github.com/istio/istio/tree/master/manifests/helm-profiles) directory. Evaluate content, and adjust compatibility mode implementation in `api/v1alpha2/compatibility_mode.go`. Check the release notes for any compatibility flags that might be not covered by the compatibilityVersion. Update `docs/user/04-00-istio-custom-resource.md` if needed. Document updated compatibility environment variables in the release notes.
- [ ] Update `k8s.io/*` and `sigs.k8s.io/*` dependencies in `go.mod` to the version used in the new Istio release.

**DoD:**
- [ ] Provide documentation.
- [ ] Test on a production-like environment.
- [ ] Verify if the solution works for both open-source Kyma and SAP BTP, Kyma runtime.
- [ ] Check the outcome of all related pipelines.
- [ ] As a PR reviewer, verify code coverage and evaluate if it is acceptable.
- [ ] Add release notes.

**Attachments**
{Link to the Istio release announcement from [Istio Release Announcements](https://istio.io/latest/news/releases/)}
{Link to the Istio upgrade notes from the announcement}

<!-- Estimation: 
Patch version update: 2
Minor version update: 3
-->
