// Copyright Nitric Pty Ltd.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package azure

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	maxLenProject    = 10
	maxLenDeployment = 10
)

type ResouceType struct {
	Abbreviation   string
	MaxLen         int
	AllowUpperCase bool
	AllowHyphen    bool
	UseName        bool
}

// https://docs.microsoft.com/en-us/azure/cloud-adoption-framework/ready/azure-best-practices/resource-abbreviations
// https://docs.microsoft.com/en-us/azure/azure-resource-manager/management/resource-name-rules
var (
	alphanumeric = regexp.MustCompile("[^a-zA-Z0-9-]+")

	// Alphanumerics, underscores, parentheses, hyphens, periods, and unicode characters that match the regex documentation.
	// Can't end with period. Regex pattern: ^[-\w\._\(\)]+$
	ResourceGroupRT = ResouceType{Abbreviation: "rg", MaxLen: 90, AllowUpperCase: true, AllowHyphen: true}

	ContainerAppRT = ResouceType{Abbreviation: "app", MaxLen: 64, UseName: true, AllowHyphen: true}
	// Alphanumerics
	RegistryRT = ResouceType{Abbreviation: "cr", MaxLen: 50, AllowUpperCase: true}
	// Alphanumerics and hyphens. Start and end with alphanumeric.
	AnalyticsWorkspaceRT = ResouceType{Abbreviation: "log", MaxLen: 63, AllowHyphen: true}
	AssignmentRT         = ResouceType{Abbreviation: "assign", MaxLen: 64, UseName: true}
	// TODO find docs on this..
	KubeRT = ResouceType{Abbreviation: "kube", MaxLen: 64, AllowUpperCase: true}
	// lowercase letters, numbers, and the '-' character, and must be between 3 and 50 characters.
	CosmosDBAccountRT = ResouceType{Abbreviation: "cosmos", MaxLen: 50, AllowHyphen: true}
	// TODO find requirements
	MongoDBRT = ResouceType{Abbreviation: "mongo", MaxLen: 64, AllowUpperCase: true}
	// TODO find requirements
	MongoCollectionRT            = ResouceType{Abbreviation: "coll", MaxLen: 24, AllowUpperCase: true, UseName: true}
	ADApplicationRT              = ResouceType{Abbreviation: "aad-app", MaxLen: 64, UseName: true}
	ADServicePrincipalRT         = ResouceType{Abbreviation: "aad-sp", MaxLen: 64, UseName: true}
	ADServicePrincipalPasswordRT = ResouceType{Abbreviation: "aad-spp", MaxLen: 64, UseName: true}
	// Lowercase letters and numbers.
	StorageAccountRT = ResouceType{Abbreviation: "st", MaxLen: 24}
	// 	Lowercase letters, numbers, and hyphens.
	// Start with lowercase letter or number. Can't use consecutive hyphens.
	StorageContainerRT = ResouceType{Abbreviation: "cont", MaxLen: 63, AllowHyphen: true, UseName: true}
	// Lowercase letters, numbers, and hyphens.
	// Can't start or end with hyphen. Can't use consecutive hyphens.
	StorageQueueRT = ResouceType{Abbreviation: "qu", MaxLen: 63, AllowHyphen: true, UseName: true}

	//Alphanumerics and hyphens. Start with letter. End with letter or digit. Can't contain consecutive hyphens.
	KeyVaultRT = ResouceType{Abbreviation: "kv", MaxLen: 14, AllowUpperCase: true}

	//Alphanumerics and hyphens.
	EventGridRT = ResouceType{Abbreviation: "evgt", MaxLen: 24, AllowUpperCase: true, AllowHyphen: true, UseName: true}

	//Alphanumerics and hyphens.
	EventSubscriptionRT = ResouceType{Abbreviation: "evt-sub", MaxLen: 24, AllowUpperCase: true, AllowHyphen: true, UseName: true}

	// Alphanumerics and hyphens, Start with letter and end with alphanumeric.
	ApiRT = ResouceType{Abbreviation: "api", MaxLen: 80, AllowHyphen: true, AllowUpperCase: true}

	// Alphanumerics and hyphens, Start with letter and end with alphanumeric.
	ApiManagementRT = ResouceType{Abbreviation: "api-mgmt", MaxLen: 80, AllowHyphen: true, AllowUpperCase: true}

	// Alphanumerics and hyphens, Start with letter and end with alphanumeric.
	ApiOperationPolicyRT = ResouceType{Abbreviation: "api-op-pol", MaxLen: 80, AllowUpperCase: true, AllowHyphen: true, UseName: true}
)

const autoNameLength = 7

func stringHead(l pulumi.Log, s string, maxLen int) string {
	if len(s) <= maxLen-autoNameLength {
		return s
	}
	_ = l.Info("shortening name from '"+s+"' to '"+s[:maxLen-autoNameLength]+"'", &pulumi.LogArgs{Ephemeral: true})
	return s[:maxLen-autoNameLength]
}

func joinCamelCase(ss []string) string {
	res := ss[0]
	for i := 1; i < len(ss); i++ {
		word := ss[i]
		res += string(bytes.ToUpper([]byte{word[0]}))
		res += word[1:]
	}
	return res
}

func resourceName(ctx *pulumi.Context, name string, rt ResouceType) string {
	var parts []string

	if rt.UseName {
		parts = []string{
			stringHead(ctx.Log, name, rt.MaxLen-len(rt.Abbreviation)-1),
			rt.Abbreviation,
		}
	} else {
		deployName := strings.TrimPrefix(ctx.Stack(), ctx.Project()+"-")
		parts = []string{
			stringHead(ctx.Log, ctx.Project(), maxLenProject),
			stringHead(ctx.Log, deployName, maxLenDeployment),
			rt.Abbreviation,
		}
	}

	// first char must be a letter
	parts[0] = strings.TrimLeft(parts[0], "0123456789-")

	for px, p := range parts {
		parts[px] = alphanumeric.ReplaceAllString(p, "")
		if !rt.AllowHyphen {
			parts[px] = strings.ReplaceAll(parts[px], "-", "")
		}
	}

	var s string

	if rt.AllowHyphen {
		s = strings.Join(parts, "-")
		s = strings.ReplaceAll(s, "--", "-")
	} else if rt.AllowUpperCase {
		s = joinCamelCase(parts)
	} else {
		s = strings.Join(parts, "")
	}

	if !rt.AllowHyphen {
		s = strings.ReplaceAll(s, "-", "")
	}

	if !rt.AllowUpperCase {
		s = strings.ToLower(s)
	}

	return stringHead(ctx.Log, s, rt.MaxLen)
}
