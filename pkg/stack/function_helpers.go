package stack

import (
	"fmt"
)

const DefaulMembraneVersion = "v0.0.1-rc.3"

func (f *Function) Name() string {
	return f.name
}

func (f *Function) VersionString(s *Stack) string {
	if f.Version != "" {
		return f.Version
	}
	return DefaulMembraneVersion
}

func (f *Function) ContextDirectory() string {
	return f.contextDirectory
}

// ImageTagName returns the default image tag for a source image built from this function
// provider the provider name (e.g. aws), used to uniquely identify builds for specific providers
func (f *Function) ImageTagName(s *Stack, provider string) string {
	if f.Tag != "" {
		return f.Tag
	}
	providerString := ""
	if provider != "" {
		providerString = "-" + provider
	}
	return fmt.Sprintf("%s-%s%s", s.Name, f.Name(), providerString)
}
