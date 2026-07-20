package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Built-in resources Consul creates itself. They are never declared in config,
// so they are excluded from "only in consul" drift. This is the ACL equivalent
// of the fact that Consul KV has no system keys: it keeps OnlyInConsul meaning
// "someone added this out of band", not "Consul's own furniture". Operator
// tokens are not filtered, so a token nobody declared still surfaces as drift.
var builtinPolicyIDs = map[string]bool{
	"00000000-0000-0000-0000-000000000001": true, // global-management
	"00000000-0000-0000-0000-000000000002": true, // builtin/global-read-only
}

const anonymousTokenAccessorID = "00000000-0000-0000-0000-000000000002"

// ConsulClient is a read-only client for the Consul ACL HTTP API.
type ConsulClient struct {
	addr   string
	token  string
	client *http.Client
}

func NewConsulClient(addr, token string) *ConsulClient {
	if addr == "" {
		addr = "http://127.0.0.1:8500"
	}
	addr = strings.TrimRight(addr, "/")
	return &ConsulClient{addr: addr, token: token, client: &http.Client{}}
}

func (c *ConsulClient) get(path string, out interface{}) error {
	req, err := http.NewRequest(http.MethodGet, c.addr+path, nil)
	if err != nil {
		return err
	}
	if c.token != "" {
		req.Header.Set("X-Consul-Token", c.token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GET %s returned %d: %s", path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// ListPolicies returns all policies except Consul built-ins. The list endpoint
// does not include Rules, so PolicyRules fills them in per policy.
func (c *ConsulClient) ListPolicies() ([]consulPolicy, error) {
	var policies []consulPolicy
	if err := c.get("/v1/acl/policies", &policies); err != nil {
		return nil, err
	}
	out := make([]consulPolicy, 0, len(policies))
	for _, p := range policies {
		if builtinPolicyIDs[p.ID] {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

// PolicyRules fetches a single policy so its Rules can be compared.
func (c *ConsulClient) PolicyRules(id string) (consulPolicy, error) {
	var p consulPolicy
	if err := c.get("/v1/acl/policy/"+id, &p); err != nil {
		return consulPolicy{}, err
	}
	return p, nil
}

// ListTokens returns all tokens except Consul built-ins. Each entry already
// carries its policy links.
func (c *ConsulClient) ListTokens() ([]consulToken, error) {
	var tokens []consulToken
	if err := c.get("/v1/acl/tokens", &tokens); err != nil {
		return nil, err
	}
	out := make([]consulToken, 0, len(tokens))
	for _, t := range tokens {
		if t.AccessorID == anonymousTokenAccessorID {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}
