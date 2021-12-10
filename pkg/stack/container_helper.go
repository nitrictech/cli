package stack

import (
	"fmt"
)

func (c *Container) Name() string {
	return c.name
}

func (c *Container) ContextDirectory() string {
	return c.contextDirectory
}

// ImageTagName returns the default image tag for a source image built from this function
// provider the provider name (e.g. aws), used to uniquely identify builds for specific providers
func (c *Container) ImageTagName(s *Stack, provider string) string {
	if c.Tag != "" {
		return c.Tag
	}
	providerString := ""
	if provider != "" {
		providerString = "-" + provider
	}
	return fmt.Sprintf("%s-%s%s", s.Name, c.Name(), providerString)
}
