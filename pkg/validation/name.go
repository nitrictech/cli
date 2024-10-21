package validation

import (
	"fmt"
	"regexp"

	"github.com/iancoleman/strcase"
)

var ResourceName_Rule = &Rule{
	name: "Invalid Name",
	// TODO: Add docs link for rule when available
	docsUrl: "",
}

// create a lower kebab case only regex
var lowerKebabCase, _ = regexp.Compile("^[a-z0-9]+(-[a-z0-9])*$")

func IsValidResourceName(name string) bool {
	return lowerKebabCase.Match([]byte(name))
}

func NewResourceNameViolationError(resourceName string, resourceType string) *RuleViolationError {
	return ResourceName_Rule.newError(fmt.Sprintf("'%s' for %s try '%s'", resourceName, resourceType, strcase.ToKebab(resourceName)))
}
