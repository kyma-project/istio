package validation

import (
	"fmt"
	istioCR "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/described_errors"
)

func ValidateAuthorizers(i istioCR.Istio) described_errors.DescribedError {
	authorizersNameSet := make(map[string]bool)
	for _, authorizer := range i.Spec.Config.Authorizers {
		_, exists := authorizersNameSet[authorizer.Name]
		if exists {
			return described_errors.NewDescribedError(fmt.Errorf("%s is duplicated", authorizer.Name), "Authorizer name needs to be unique").SetWarning()
		}
		authorizersNameSet[authorizer.Name] = true
	}
	return nil
}
