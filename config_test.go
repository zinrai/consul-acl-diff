package main

import "testing"

func TestNormalizeRules(t *testing.T) {
	tests := []struct {
		name  string
		a, b  string
		equal bool
	}{
		{"crlf vs lf", "key \"x\" {\r\n  policy = \"read\"\r\n}", "key \"x\" {\n  policy = \"read\"\n}", true},
		{"trailing whitespace", "key \"x\" {  \n  policy = \"read\"  \n}  ", "key \"x\" {\n  policy = \"read\"\n}", true},
		{"internal whitespace differs", "key \"x\" { policy = \"read\" }", "key \"x\" {  policy = \"read\"  }", false},
		{"content differs", "policy = \"read\"", "policy = \"write\"", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeRules(tt.a) == normalizeRules(tt.b); got != tt.equal {
				t.Errorf("normalizeRules equality = %v, want %v", got, tt.equal)
			}
		})
	}
}

func TestStringSetEqual(t *testing.T) {
	tests := []struct {
		name  string
		a, b  []string
		equal bool
	}{
		{"same order", []string{"a", "b"}, []string{"a", "b"}, true},
		{"different order", []string{"a", "b"}, []string{"b", "a"}, true},
		{"different length", []string{"a"}, []string{"a", "b"}, false},
		{"different content", []string{"a"}, []string{"b"}, false},
		{"empty vs nil", []string{}, nil, true},
		{"duplicates matter", []string{"a", "a"}, []string{"a", "b"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stringSetEqual(tt.a, tt.b); got != tt.equal {
				t.Errorf("stringSetEqual = %v, want %v", got, tt.equal)
			}
		})
	}
}

func TestComparePolicy(t *testing.T) {
	desired := Policy{
		Name:        "p",
		Description: "d",
		Rules:       "key \"x\" {\n  policy = \"read\"\n}",
		Datacenters: []string{"dc1"},
	}
	current := consulPolicy{
		Name:        "p",
		Description: "d",
		Rules:       "key \"x\" {\r\n  policy = \"read\"\r\n}",
		Datacenters: []string{"dc1"},
	}

	if _, changed := comparePolicy(current, desired); changed {
		t.Error("identical policy (crlf only) should not be reported as changed")
	}

	descChanged := current
	descChanged.Description = "different"
	change, changed := comparePolicy(descChanged, desired)
	if !changed || !change.DescChanged || change.RulesChanged {
		t.Errorf("description change: got %+v changed=%v", change, changed)
	}
	if change.CurrentDesc != "different" || change.DesiredDesc != "d" {
		t.Errorf("description change should record both sides: %+v", change)
	}

	rulesChanged := current
	rulesChanged.Rules = "key \"x\" {\n  policy = \"write\"\n}"
	if change, changed := comparePolicy(rulesChanged, desired); !changed || !change.RulesChanged {
		t.Errorf("rules change: got %+v changed=%v", change, changed)
	}

	dcChanged := current
	dcChanged.Datacenters = []string{"dc2"}
	if change, changed := comparePolicy(dcChanged, desired); !changed || !change.DatacentersChanged {
		t.Errorf("datacenter change: got %+v changed=%v", change, changed)
	}
}

func TestCompareToken(t *testing.T) {
	desired := Token{AccessorID: "a", Description: "web", Policies: []string{"p1", "p2"}}
	current := consulToken{
		AccessorID:  "a",
		Description: "web",
		Policies:    []consulPolicyLink{{Name: "p2"}, {Name: "p1"}},
	}

	if _, changed := compareToken(current, desired); changed {
		t.Error("same policy set in different order should not be reported as changed")
	}

	descChanged := current
	descChanged.Description = "different"
	if change, changed := compareToken(descChanged, desired); !changed || !change.DescChanged {
		t.Errorf("description change: got %+v changed=%v", change, changed)
	}

	policiesChanged := current
	policiesChanged.Policies = []consulPolicyLink{{Name: "p1"}}
	change, changed := compareToken(policiesChanged, desired)
	if !changed || !change.PoliciesChanged {
		t.Errorf("policy set change: got %+v changed=%v", change, changed)
	}
	// CurrentPolicies and DesiredPolicies are recorded sorted for display.
	if len(change.DesiredPolicies) != 2 || change.DesiredPolicies[0] != "p1" || change.DesiredPolicies[1] != "p2" {
		t.Errorf("desired policies should be sorted: %+v", change.DesiredPolicies)
	}
}
