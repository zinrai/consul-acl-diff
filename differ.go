package main

import (
	"fmt"
	"sort"
	"strings"
)

// CalculateDiff compares the config against the live Consul state.
func CalculateDiff(client *ConsulClient, cfg *Config) (*DiffResult, error) {
	diff := &DiffResult{}
	if err := diffPolicies(client, cfg, diff); err != nil {
		return nil, err
	}
	if err := diffTokens(client, cfg, diff); err != nil {
		return nil, err
	}
	return diff, nil
}

func diffPolicies(client *ConsulClient, cfg *Config, diff *DiffResult) error {
	consulPolicies, err := client.ListPolicies()
	if err != nil {
		return fmt.Errorf("failed to list policies: %w", err)
	}
	byName := make(map[string]consulPolicy, len(consulPolicies))
	for _, p := range consulPolicies {
		byName[p.Name] = p
	}

	localNames := make(map[string]bool, len(cfg.Policies))
	for _, local := range cfg.Policies {
		localNames[local.Name] = true

		current, ok := byName[local.Name]
		if !ok {
			diff.PolicyOnlyInLocal = append(diff.PolicyOnlyInLocal,
				Ref{Key: local.Name, Description: local.Description})
			continue
		}

		// Rules are absent from the list response, so fetch the full policy.
		full, err := client.PolicyRules(current.ID)
		if err != nil {
			return fmt.Errorf("failed to read policy %q: %w", local.Name, err)
		}
		if change, changed := comparePolicy(full, local); changed {
			diff.PolicyModified = append(diff.PolicyModified, change)
		}
	}

	for _, p := range consulPolicies {
		if !localNames[p.Name] {
			diff.PolicyOnlyInConsul = append(diff.PolicyOnlyInConsul,
				Ref{Key: p.Name, Description: p.Description})
		}
	}
	return nil
}

func comparePolicy(current consulPolicy, desired Policy) (PolicyChange, bool) {
	change := PolicyChange{Name: desired.Name}
	changed := false

	if current.Description != desired.Description {
		change.DescChanged = true
		change.CurrentDesc = current.Description
		change.DesiredDesc = desired.Description
		changed = true
	}
	if normalizeRules(current.Rules) != normalizeRules(desired.Rules) {
		change.RulesChanged = true
		changed = true
	}
	if !stringSetEqual(current.Datacenters, desired.Datacenters) {
		change.DatacentersChanged = true
		changed = true
	}
	return change, changed
}

func diffTokens(client *ConsulClient, cfg *Config, diff *DiffResult) error {
	consulTokens, err := client.ListTokens()
	if err != nil {
		return fmt.Errorf("failed to list tokens: %w", err)
	}
	byAccessor := make(map[string]consulToken, len(consulTokens))
	for _, t := range consulTokens {
		byAccessor[t.AccessorID] = t
	}

	localAccessors := make(map[string]bool, len(cfg.Tokens))
	for _, local := range cfg.Tokens {
		localAccessors[local.AccessorID] = true

		current, ok := byAccessor[local.AccessorID]
		if !ok {
			diff.TokenOnlyInLocal = append(diff.TokenOnlyInLocal,
				Ref{Key: local.AccessorID, Description: local.Description})
			continue
		}
		if change, changed := compareToken(current, local); changed {
			diff.TokenModified = append(diff.TokenModified, change)
		}
	}

	for _, t := range consulTokens {
		if !localAccessors[t.AccessorID] {
			diff.TokenOnlyInConsul = append(diff.TokenOnlyInConsul,
				Ref{Key: t.AccessorID, Description: t.Description})
		}
	}
	return nil
}

func compareToken(current consulToken, desired Token) (TokenChange, bool) {
	change := TokenChange{AccessorID: desired.AccessorID}
	changed := false

	if current.Description != desired.Description {
		change.DescChanged = true
		change.CurrentDesc = current.Description
		change.DesiredDesc = desired.Description
		changed = true
	}

	currentPolicies := policyLinkNames(current.Policies)
	if !stringSetEqual(currentPolicies, desired.Policies) {
		change.PoliciesChanged = true
		change.CurrentPolicies = sortedCopy(currentPolicies)
		change.DesiredPolicies = sortedCopy(desired.Policies)
		changed = true
	}
	return change, changed
}

func policyLinkNames(links []consulPolicyLink) []string {
	names := make([]string, 0, len(links))
	for _, l := range links {
		names = append(names, l.Name)
	}
	return names
}

// stringSetEqual reports whether two slices hold the same multiset of strings,
// ignoring order.
func stringSetEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[string]int, len(a))
	for _, s := range a {
		counts[s]++
	}
	for _, s := range b {
		counts[s]--
		if counts[s] < 0 {
			return false
		}
	}
	return true
}

func sortedCopy(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	sort.Strings(out)
	return out
}

// PrintDiff writes the diff in a human-readable form.
func PrintDiff(diff *DiffResult) {
	if !diff.HasDrift() {
		fmt.Println("No drift. Config matches Consul.")
		return
	}
	printPolicySection(diff)
	printTokenSection(diff)
	printSummary(diff)
}

func printPolicySection(diff *DiffResult) {
	if len(diff.PolicyModified) == 0 &&
		len(diff.PolicyOnlyInLocal) == 0 &&
		len(diff.PolicyOnlyInConsul) == 0 {
		return
	}

	fmt.Println("Policies")
	for _, c := range diff.PolicyModified {
		fmt.Printf("  ~ %s (modified)\n", c.Name)
		if c.DescChanged {
			fmt.Printf("      description: %q → %q\n", c.CurrentDesc, c.DesiredDesc)
		}
		if c.RulesChanged {
			fmt.Println("      rules: changed")
		}
		if c.DatacentersChanged {
			fmt.Println("      datacenters: changed")
		}
	}
	for _, r := range diff.PolicyOnlyInLocal {
		fmt.Printf("  + %s (only in local, not yet applied)\n", r.Key)
	}
	for _, r := range diff.PolicyOnlyInConsul {
		fmt.Printf("  - %s (only in consul, unmanaged)\n", r.Key)
	}
	fmt.Println()
}

func printTokenSection(diff *DiffResult) {
	if len(diff.TokenModified) == 0 &&
		len(diff.TokenOnlyInLocal) == 0 &&
		len(diff.TokenOnlyInConsul) == 0 {
		return
	}

	fmt.Println("Tokens")
	for _, c := range diff.TokenModified {
		fmt.Printf("  ~ %s (modified)\n", c.AccessorID)
		if c.DescChanged {
			fmt.Printf("      description: %q → %q\n", c.CurrentDesc, c.DesiredDesc)
		}
		if c.PoliciesChanged {
			fmt.Printf("      policies: [%s] → [%s]\n",
				strings.Join(c.CurrentPolicies, ", "), strings.Join(c.DesiredPolicies, ", "))
		}
	}
	for _, r := range diff.TokenOnlyInLocal {
		fmt.Printf("  + %s (only in local, not yet applied)\n", labelRef(r))
	}
	for _, r := range diff.TokenOnlyInConsul {
		fmt.Printf("  - %s (only in consul, unmanaged)\n", labelRef(r))
	}
	fmt.Println()
}

// labelRef annotates an opaque accessor id with its description when present.
func labelRef(r Ref) string {
	if r.Description != "" {
		return fmt.Sprintf("%s %q", r.Key, r.Description)
	}
	return r.Key
}

func printSummary(diff *DiffResult) {
	fmt.Println("Drift detected.")
	fmt.Printf("  Policies: %d modified, %d only in local, %d only in consul\n",
		len(diff.PolicyModified), len(diff.PolicyOnlyInLocal), len(diff.PolicyOnlyInConsul))
	fmt.Printf("  Tokens:   %d modified, %d only in local, %d only in consul\n",
		len(diff.TokenModified), len(diff.TokenOnlyInLocal), len(diff.TokenOnlyInConsul))
}
