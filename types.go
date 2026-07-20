package main

// Config is the YAML configuration, identical to the file consul-acl-sync
// applies. consul-acl-diff only reads it.
type Config struct {
	Policies []Policy `yaml:"policies"`
	Tokens   []Token  `yaml:"tokens"`
}

// Policy is a Consul ACL policy, keyed by Name.
type Policy struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Rules       string   `yaml:"rules"`
	Datacenters []string `yaml:"datacenters"`
}

// Token is a Consul ACL token, keyed by AccessorID. SecretID is parsed so the
// config loads cleanly, but it is never compared or transmitted: a token secret
// is immutable after creation and this tool only reads.
type Token struct {
	AccessorID  string   `yaml:"accessor_id"`
	SecretID    string   `yaml:"secret_id"`
	Description string   `yaml:"description"`
	Policies    []string `yaml:"policies"`
}

// consulPolicy is the subset of the Consul policy API we read. The list
// endpoint omits Rules, so it is filled in per policy on demand.
type consulPolicy struct {
	ID          string   `json:"ID"`
	Name        string   `json:"Name"`
	Description string   `json:"Description"`
	Rules       string   `json:"Rules"`
	Datacenters []string `json:"Datacenters"`
}

// consulToken is the subset of the Consul token API we read. The list endpoint
// already carries the policy links, so no per-token fetch is needed.
type consulToken struct {
	AccessorID  string             `json:"AccessorID"`
	Description string             `json:"Description"`
	Policies    []consulPolicyLink `json:"Policies"`
}

type consulPolicyLink struct {
	ID   string `json:"ID"`
	Name string `json:"Name"`
}

// Ref identifies a resource that exists on only one side of the diff. Key is
// the policy name or the token accessor_id. Description is carried for display,
// since accessor ids are opaque.
type Ref struct {
	Key         string
	Description string
}

// PolicyChange records how a policy present on both sides differs.
type PolicyChange struct {
	Name               string
	DescChanged        bool
	CurrentDesc        string
	DesiredDesc        string
	RulesChanged       bool
	DatacentersChanged bool
}

// TokenChange records how a token present on both sides differs.
type TokenChange struct {
	AccessorID      string
	DescChanged     bool
	CurrentDesc     string
	DesiredDesc     string
	PoliciesChanged bool
	CurrentPolicies []string
	DesiredPolicies []string
}

// DiffResult is the symmetric diff between config and Consul.
type DiffResult struct {
	PolicyOnlyInLocal  []Ref
	PolicyOnlyInConsul []Ref
	PolicyModified     []PolicyChange

	TokenOnlyInLocal  []Ref
	TokenOnlyInConsul []Ref
	TokenModified     []TokenChange
}

// HasDrift reports whether config and Consul differ in any way.
func (d *DiffResult) HasDrift() bool {
	return len(d.PolicyOnlyInLocal) > 0 ||
		len(d.PolicyOnlyInConsul) > 0 ||
		len(d.PolicyModified) > 0 ||
		len(d.TokenOnlyInLocal) > 0 ||
		len(d.TokenOnlyInConsul) > 0 ||
		len(d.TokenModified) > 0
}
