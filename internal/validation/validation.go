package validation

import (
	"fmt"
	"regexp"

	istioCR "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/describederrors"
)

func ValidateAuthorizers(i istioCR.Istio) describederrors.DescribedError {
	authorizersNameSet := make(map[string]bool)
	for _, authorizer := range i.Spec.Config.Authorizers {
		_, exists := authorizersNameSet[authorizer.Name]
		if exists {
			return describederrors.NewDescribedError(fmt.Errorf("%s is duplicated", authorizer.Name), "Authorizer name needs to be unique").SetWarning()
		}
		authorizersNameSet[authorizer.Name] = true
	}
	return nil
}

func ValidateProxyStatsMatcher(i istioCR.Istio) describederrors.DescribedError {
	if i.Spec.Config.ProxyStatsMatcher == nil {
		return nil
	}
	for _, r := range i.Spec.Config.ProxyStatsMatcher.InclusionRegexps {
		if _, err := regexp.Compile(r); err != nil {
			return describederrors.NewDescribedError(fmt.Errorf("%q is not a valid regexp: %w", r, err), "ProxyStatsMatcher inclusionRegexps contains an invalid regular expression")
		}
	}
	return nil
}
