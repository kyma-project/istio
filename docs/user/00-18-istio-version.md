# Istio Version

The version of Istio depends on the version of the Istio module that you use. If a new version of Istio module introduces a new version of Istio, an upgrade of the module automatically (triggers) an upgrade of Istio.

The latest release includes the following versions of Istio and Envoy:  

**Istio version:** 1.22.3

**Envoy version:** 1.30.5

## Upgrades and Downgrades
//czy to się nadaje na hp? przeciez skr nie ma wyboru jaką wersję Istio instaluje i upgrades dzieją się automatycznie, nie?

You can only skip a version of the Istio module if the difference between the minor version of Istio it contains and the minor version of Istio you're using is not greater than one (for example, 1.2.3 -> 1.3.0).
If the difference is greater than one minor version (for example, 1.2.3 -> 1.4.0), the reconciliation fails.
The same happens if you try to update the major version (for example, 1.2.3 -> 2.0.0) or downgrade the version. 
Such scenarios are not supported and cause the Istio CR to be in the `Warning` state with the `Ready` condition set to `false` and the reason being `IstioVersionUpdateNotAllowed`.

## Compatibility Mode

The Istio module applies an opinionated subset of Istio compatibilityVersion variables, and supports compatibility with the previous minor version of Istio. For example, the Istio module with Istio 1.21.0 applies a compatibility version of Istio 1.20. See [Compatibility Versions](https://istio.io/latest/docs/setup/additional-setup/compatibility-versions/).

To enable compatibility mode in the Istio module, you can set the **spec.compatibilityMode** field in the Istio CR to `true`. This allows you to mitigate breaking changes when a new release introduces an Istio upgrade.


The following Istio Pilot environment variables are applied when you set `spec.compatibilityMode: true` in Istio CR:

Name                                   | Value
---------------------------------------|--------
**ENABLE_ENHANCED_RESOURCE_SCOPING**   | `false`
**ENABLE_RESOLUTION_NONE_TARGET_PORT** | `false`

The following Istio Proxy environment variable is applied when you set `spec.compatibilityMode: true` in Istio CR:

Name                | Value
--------------------|--------
**ISTIO_DELTA_XDS** | `false`


> [!WARNING]
> You can use the compatibility mode to retain the behavior of the current Istio version when a new version of the Istio module with a higher version of Istio is released. Then, the compatibility will be first set to a minor version lower than the one you are currently using. If this lower version’s behavior is not compatible with your current mesh setup, some configurations may be broken until the new release of the Istio module is rolled out.