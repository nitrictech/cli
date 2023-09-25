package validation

import (
	"errors"
	"regexp"
)

// StringValidator is a function that returns an error if the input is invalid.
type StringValidator func(string) error

func NotBlankValidator(message string) StringValidator {
	return func(value string) error {
		if value == "" {
			return errors.New(message)
		}
		return nil
	}
}

func RegexValidator(regex *regexp.Regexp, message string) StringValidator {
	return func(value string) error {
		if !regex.MatchString(value) {
			return errors.New(message)
		}

		return nil
	}
}

func ComposeValidators(validators ...StringValidator) StringValidator {
	return func(value string) error {
		for _, v := range validators {
			if err := v(value); err != nil {
				return err
			}
		}
		return nil
	}
}

// var alphanumOnly = regexValidator(nameRegex, "name must only contain letters, numbers and dashes")
