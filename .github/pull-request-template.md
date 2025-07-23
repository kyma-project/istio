<!-- Thank you for your contribution. Before you submit the pull request:
1. Follow contributing guidelines, templates, the recommended Git workflow, and any related documentation.
2. Read and submit the required Contributor Licence Agreements (https://github.com/kyma-project/community/blob/main/docs/contributing/02-contributing.md#agreements-and-licenses).
3. Test your changes and attach their results to the pull request.
4. Update the relevant documentation.
-->

**Description**

Changes proposed in this pull request:

- ...

**Pre-Merge Checklist**

Consider all the following items. If any of them are violated, or you are not sure about them, please add a comment to the PR.

- [ ] Verify code coverage and evaluate if it is acceptable.
- [ ] Create RN for introduced changes.
- [ ] If Kubebuilder changes were made, run `make generate-manifests` and commit the changes before merge.
- [ ] Ensure whether pre-existing managed resources are correctly handled.
- [ ] Change works on all hyperscalers supported by BTP Kyma Runtime.
- [ ] There is no upgrade downtime.
- [ ] For infrastructure changes, check the cost in the hyperscaler.
- [ ] RBAC settings are as restrictive as possible.
- [ ] If any new libraries are added, verify license compliance and maintainability and make a comment in the PR with details. We only allow to include stable releases into the project.
- [ ] If this change should be cherry-picked to active release branches.
- [ ] If the change of the configuration does not introduce any additional latency.

**Related issues**
<!-- If you refer to a particular issue, provide its number. For example, `Resolves #123`, `Fixes #43`, or `See also #33`. -->
