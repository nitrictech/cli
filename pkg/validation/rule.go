package validation

import (
	"errors"
	"fmt"
)

type Rule struct {
	name    string
	docsUrl string
}

func (r *Rule) newError(message string) *RuleViolationError {
	return &RuleViolationError{
		rule:    r,
		message: message,
	}
}

func (r *Rule) String() string {
	return fmt.Sprintf("%s: %s", r.name, r.docsUrl)
}

type RuleViolationError struct {
	rule    *Rule
	message string
}

func (r *RuleViolationError) Error() string {
	return fmt.Sprintf("%s: %s", r.rule.name, r.message)
}

func GetRuleViolation(err error) *Rule {
	ruleViolation := &RuleViolationError{}

	if errors.As(err, &ruleViolation) {
		return ruleViolation.rule
	}

	return nil
}
