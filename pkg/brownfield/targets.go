// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"strings"

	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
)

// Target uniquely identifies a subset of App Gateway configuration, which AGIC will manage or be prohibited from managing.
type Target struct {
	Hostname string `json:"Hostname,omitempty"`
	Path     string `json:"Path,omitempty"`
}

// IsBlacklisted figures out whether a given Target objects in a list of blacklisted targets.
func (t Target) IsBlacklisted(blacklist *[]Target) bool {
	for _, blTarget := range *blacklist {

		// An empty blacklist hostname indicates that any hostname would be blacklisted.
		// If host names match - this target is in the blacklist.
		// AGIC is allowed to create and modify App Gwy config for blank host.
		hostIsSame := blTarget.Hostname == "" || strings.ToLower(t.Hostname) == strings.ToLower(blTarget.Hostname)

		pathIsSame := blTarget.Path == "" || strings.ToLower(t.Path) == strings.ToLower(blTarget.Path)

		// With this version we keep things as simple as possible: match host and exact path to determine
		// whether given target is in the blacklist. Ideally this would be URL Path set overlap operation,
		// which we deliberately leave for a later time.
		if hostIsSame && pathIsSame {
			return true // Found it
		}
	}
	return false // Did not find it
}

// prettyTarget is used for pretty-printing the Target struct for debugging purposes.
type prettyTarget struct {
	Hostname string `json:"Hostname"`
	Path     string `json:"Path,omitempty"`
}

// GetTargetBlacklist returns the list of Targets given a list ProhibitedTarget CRDs.
func GetTargetBlacklist(prohibitedTargets []*ptv1.AzureIngressProhibitedTarget) TargetBlacklist {
	var target []Target
	for _, prohibitedTarget := range prohibitedTargets {
		if len(prohibitedTarget.Spec.Paths) == 0 {
			target = append(target, Target{
				Hostname: prohibitedTarget.Spec.Hostname,
			})
		}
		for _, path := range prohibitedTarget.Spec.Paths {
			target = append(target, Target{
				Hostname: prohibitedTarget.Spec.Hostname,
				Path:     strings.ToLower(path),
			})
		}
	}
	return &target
}
