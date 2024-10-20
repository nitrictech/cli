package validation

import (
	"fmt"
	"regexp"
)

var ResourceName_Rule = &Rule{
	name:    "Invalid Resource Name",
	docsUrl: "https://docs.example.com/validation/resource-name",
}

// create a lower kebab case only regex
var lowerKebabCase, _ = regexp.Compile("^[a-z0-9]+(-[a-z0-9])*$")

func IsValidResourceName(name string) bool {
	return lowerKebabCase.Match([]byte(name))
}

func NewResourceNameViolationError(resourceName string, resourceType string) *RuleViolationError {
	return ResourceName_Rule.newError(fmt.Sprintf("'%s' for resource %s", resourceName, resourceType))
}
