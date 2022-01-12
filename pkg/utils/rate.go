package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// RateToCron - Converts a valid rate expression
// into a simple crontab expression
func RateToCron(rate string) (string, error) {
	rateParts := strings.Split(rate, " ")

	rateNum := rateParts[0]
	rateType := rateParts[1]

	num, err := strconv.Atoi(rateNum)

	if err != nil {
		return "", fmt.Errorf("invalid rate expression %s; %v", rate, err)
	}

	switch rateType {
	// Every nth minute
	case "minutes":
		return fmt.Sprintf("*/%d * * * *", num), nil
	case "hours":
		// The top of every nth hour
		return fmt.Sprintf("0 */%d * * *", num), nil
	case "days":
		// Midnight every nth day
		return fmt.Sprintf("0 0 */%d * *", num), nil
	default:
		return "", fmt.Errorf("invalid rate expression %s; %s must be one of [minutes, hours, days]", rate, rateType)
	}
}
