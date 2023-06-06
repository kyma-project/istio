package testcontext

import (
	"github.com/cucumber/godog"
)

const uninstallTag = "@uninstall"

func HasUninstallTag(s *godog.Scenario) bool {
	for _, tag := range s.Tags {
		if tag.Name == uninstallTag {
			return true
		}
	}
	return false
}
