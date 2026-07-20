# consul-acl-diff

Read-only drift detection for Consul ACL policies and tokens against a YAML
configuration. This is the verify side of
[consul-acl-sync](https://github.com/zinrai/consul-acl-sync): consul-acl-sync
writes, consul-acl-diff only reads and reports.

## Usage

```bash
$ consul-acl-diff -config config.yaml
```

The config is the same file consul-acl-sync applies.

See `example.yaml` for the schema.

The Consul address defaults to `http://127.0.0.1:8500`. Point it elsewhere with
`-consul-addr`:

```bash
$ consul-acl-diff -config config.yaml -consul-addr http://consul.example.com:8500
```

## Output

Drift is reported in three categories per resource:

- `modified`: exists on both sides but differs (policy rules, description, or
  datacenters, or token description or policy set)
- `only in local`: declared in config but absent from Consul, so consul-acl-sync
  would create it
- `only in consul`: present in Consul but not declared, so it was created out of
  band

## Exit codes

Matching consul-kv-diff, so the tool works as a CI gate:

- `0`: config matches Consul
- `1`: drift found, or an error occurred

All three categories, including `only in consul`, count as drift.

## Identity and comparison

- Policies are keyed by `name`.
- Tokens are keyed by `accessor_id`. `secret_id` is never compared. A token
  secret is immutable after creation and this tool never reads or transmits it.
- Rules are compared after whitespace normalization, matching consul-acl-sync,
  so cosmetic differences are not reported as drift.

## Built-in resources

Consul creates a few resources itself: the `global-management` and
`builtin/global-read-only` policies, and the anonymous token. These are excluded
from `only in consul`, the same way Consul KV has no system keys for
consul-kv-diff to trip over.

Operator-created tokens are not filtered. A token that nobody declared in the
config, including the bootstrap management token, surfaces as `only in consul`
drift. Bring it under management by pinning its `accessor_id` and `secret_id` in
the config, or accept the drift signal.

## ACL token

The token is read from the `CONSUL_HTTP_TOKEN` environment variable, following
the `consul` CLI convention, rather than a flag so it does not leak into process
listings or shell history. It needs `acl:read` on a cluster that enforces ACLs.

## License

This project is licensed under the [MIT License](./LICENSE).
