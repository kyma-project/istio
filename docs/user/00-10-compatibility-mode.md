# Compatibility Mode
To revert certain changes in Istio's behavior when you encounter compatibility issues with its new version, consider enabling compatibility mode.

Compatibility mode allows you to revert certain changes in Istio's behavior, and it is recommended only when you encounter compatibility issues with the new version of Istio. The Istio module supports compatibility with the previous minor version of Istio. For example, for the version of the Istio module that contains Istio 1.21, you can apply a compatibility version of Istio 1.20. See [Compatibility Versions](https://istio.io/latest/docs/setup/additional-setup/compatibility-versions/).

> [!WARNING]
> You can use the compatibility mode to retain the behavior of the current Istio version before a new version of the Istio module with a higher version of Istio is released. Then, the compatibility is first set to a minor version lower than the one you are currently using. If this lower version’s behavior is not compatible with your current mesh setup, some configurations may be broken until the new release of the Istio module is rolled out.

To enable compatibility mode, set the **spec.compatibilityMode** field in the Istio CR to `true`.

When you set `spec.compatibilityMode: true`, the Istio module applies an opinionated subset of Istio **compatibilityVersion** variables. The compatibility version of Istio 1.22 includes the following Istio Pilot and Istio Proxy environment variables:

| Istio Component | Name                                 | Value   |
|-----------------|--------------------------------------|---------|
| Istio Pilot     | **ENABLE_DELIMITED_STATS_TAG_REGEX** | `false` |
| Istio Proxy     | **ENABLE_DEFERRED_CLUSTER_CREATION** | `false` |
| Istio Proxy     | **ENABLE_DELIMITED_STATS_TAG_REGEX** | `false` |

To learn more about the changes that specific compatibility versions revert, follow the [Istio release notes](https://github.com/kyma-project/istio/releases).