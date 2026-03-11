# Relay Observability and Compatibility

**Status:** Draft
**Author:** Architecture note 2026-03-10

## Summary

Before changing message rendering or introducing more aggressive client/server compatibility rules, the relay needs to become a better source of truth for what is actually deployed and actively used.

The first priority is to have yepanywhere servers send version and protocol metadata when they register with the relay. That metadata must be optional so newer servers can report it immediately without breaking older servers or older hosted clients.

This note defines:

1. The immediate registration metadata change
2. The observability data the relay should record
3. The rollout constraints for backward compatibility
4. The next phases for capabilities, upgrade messaging, and eventual deprecation policy

## Why This Comes First

Right now the relay can report only instantaneous in-memory connection counts:

- `waiting` and `pairs` from [packages/relay/src/server.ts](../../packages/relay/src/server.ts)
- tracked by the waiting map and pair set in [packages/relay/src/connections.ts](../../packages/relay/src/connections.ts)

That is useful for debugging a live incident, but it is not enough for rollout planning.

Missing information:

- Which yepanywhere versions are currently connecting to relay
- Which protocol versions are active among real remote users
- How many distinct installs are using relay in the last 7/14/30 days
- Whether a future hosted frontend change would strand phone users on older servers

Without that, any compatibility cutoff is guesswork.

## Current State

Current relay registration is intentionally minimal:

```ts
interface RelayServerRegister {
  type: "server_register";
  username: string;
  installId: string;
}
```

See [packages/shared/src/relay-protocol.ts](../../packages/shared/src/relay-protocol.ts).

The server already exposes useful compatibility information on `/version`:

- `current`
- `resumeProtocolVersion`
- `capabilities`

See [packages/server/src/routes/version.ts](../../packages/server/src/routes/version.ts).

That means the missing piece is not inventing compatibility data. It is carrying a compatible subset of that identity into the relay path and recording it there.

## Goals

1. Keep relay registration backward compatible
2. Make the relay the source of truth for remote-active installs
3. Support future hosted frontend migrations without blind cutovers
4. Create a telemetry foundation that still works if relay becomes multi-node

## Non-Goals

- No immediate hard compatibility break
- No mandatory global telemetry for all installs
- No attempt in this phase to solve multi-relay consensus
- No attempt in this phase to redesign message rendering

## Compatibility Identity

We should define one app-owned compatibility identity shape for the server:

```ts
interface ServerCompatibilityIdentity {
  installId: string;
  appVersion?: string;
  resumeProtocolVersion?: number;
  renderProtocolVersion?: number;
  capabilities?: string[];
}
```

Notes:

- `installId` already exists and remains required for relay registration ownership
- Everything else starts optional for backward compatibility
- `renderProtocolVersion` is included now even if the renderer migration is deferred; it gives us a stable place to version future render contracts
- `capabilities` are for feature detection, not for hard compatibility decisions

This identity should be serializable in relay registration and conceptually align with `/version`, even if the exact types are not yet shared from a single module.

## Phase 1: Optional Relay Registration Metadata

Extend `RelayServerRegister` with optional metadata:

```ts
interface RelayServerRegister {
  type: "server_register";
  username: string;
  installId: string;
  appVersion?: string;
  resumeProtocolVersion?: number;
  renderProtocolVersion?: number;
  capabilities?: string[];
}
```

Requirements:

1. Older servers must continue to register successfully without the new fields
2. Newer relays must accept and store the fields when present
3. Older relays should ignore unknown fields naturally
4. Validation should be permissive enough to avoid accidental rejection when fields are absent

This is the highest-priority infrastructure change because it gives immediate visibility with minimal protocol risk.

## Relay Telemetry Model

The relay should record append-only structured events, similar in spirit to the simple update server, but focused on actual relay activity rather than version checks.

Recommended initial events:

- `server_register`
- `server_disconnect`
- `client_connect_success`
- `client_connect_error`

Recommended fields on `server_register`:

- `timestamp`
- `installId`
- `username`
- `appVersion`
- `resumeProtocolVersion`
- `renderProtocolVersion`
- `capabilities`
- `relayNodeId` or hostname

Recommended fields on connection events:

- `timestamp`
- `username`
- `relayNodeId`
- `reason` for failures or disconnects when available

The key distinction is:

- Relay telemetry answers "who is actively using remote access"
- Update checks answer "who looked for a newer npm version"

For migration planning, relay telemetry is more important.

## Metrics and Dashboards

The relay stats page does not need to be elaborate. A simple operational view is enough.

Recommended views:

1. Unique remote-active installs per day by `appVersion`
2. Unique remote-active installs per day by `renderProtocolVersion`
3. Successful client connections per day
4. Instantaneous or sampled `waiting` and `pairs` counts over time
5. Optional top-level counts for the trailing 7/14/30 days

This is enough to answer:

- Are older servers still actively used from phones?
- Would a hosted frontend change break real users?
- Was a spike in waiting connections real or just stale state that disappeared on restart?

## Why Waiting Count Can Be Misleading Today

Today `waiting` is just the count of registered idle server sockets in memory. It is not a durable count of installs.

Important details:

- The value comes directly from `waiting.size` in [packages/relay/src/connections.ts](../../packages/relay/src/connections.ts)
- It is exposed directly on `/health` in [packages/relay/src/server.ts](../../packages/relay/src/server.ts)
- Waiting sockets are cleaned up on normal close and relay ping/pong timeout in [packages/relay/src/ws-handler.ts](../../packages/relay/src/ws-handler.ts)

That means a restart can absolutely reduce the number sharply if some of the prior state was stale or if not all servers reconnect immediately.

This is another reason to add event logging and time-series sampling rather than relying on one health number.

## Backward Compatibility Policy

Hosted remote access is a core feature, so this work should default to compatibility-first rollout.

Policy:

1. Do not require a server upgrade just to keep using hosted `/remote`
2. Prefer dual-read compatibility in the hosted client whenever protocol evolution allows it
3. Use protocol versions for hard cutoffs only after observing actual relay usage
4. Treat capability flags as advisory, not as the sole basis for deprecation

In practical terms:

- Relay registration metadata is optional first
- Relay begins measuring protocol/version adoption before any cutoff discussion
- Hosted client warnings can be added later, but should not block access until there is clear usage data and a deliberate deprecation decision

## Multi-Relay Considerations

This observability work should be designed so it still makes sense in a multi-relay setup.

That means:

- Every event should identify the relay node that observed it
- Aggregation should work across nodes
- Telemetry should not depend on strong inter-relay consistency

This is intentionally looser than username ownership coordination. Telemetry can remain eventually consistent and still be useful.

## Future Phases

### Phase 2: Shared Compatibility Identity

Reduce duplication between `/version` and relay registration by defining a shared server compatibility identity in shared code.

This should make it easier to:

- keep version/protocol reporting consistent
- expose the same compatibility data in settings, logs, and relay metrics
- avoid one-off protocol fields proliferating in multiple places

### Phase 3: Capability Reporting

Once basic version/protocol telemetry is in place, expand capability reporting so the relay and hosted client can reason about feature support more clearly.

Examples:

- `deviceBridge`
- future render capabilities
- remote transport capabilities

Capabilities should remain feature-level hints. Protocol version fields should continue to carry hard compatibility meaning.

### Phase 4: Upgrade Signaling

After telemetry exists, add softer upgrade infrastructure:

- warnings in hosted remote UI when the server is missing recommended protocol/features
- optional settings-page visibility into whether the server is behind current hosted expectations
- later, possibly a more guided update flow if the product can safely support it

This phase should avoid creating a phone-only dead end where the user is told to update but has no practical way to do so.

### Phase 5: Deprecation Policy

Only after adoption data exists should we define a real cutoff policy for older relay/server protocol versions.

That policy should be based on observed remote-active installs over a recent trailing window, not only npm release timing.

Example decision inputs:

- active installs using relay in the last 14 days by protocol version
- successful remote connections by protocol version
- estimated blast radius of dropping a legacy path

## Proposed Immediate Work

1. Extend `RelayServerRegister` with optional version and protocol fields
2. Update the yepanywhere server relay client to send those fields when available
3. Update the relay to log append-only registration and connection events
4. Add a lightweight stats page showing version/protocol adoption and connection counts

This is the minimum useful slice. It improves observability immediately and sets up future compatibility work without forcing a renderer migration or a hard break.

## Open Questions

1. Should `appVersion` come directly from the existing `/version` route logic or from a lower-level shared utility?
2. Should `renderProtocolVersion` be introduced now as a placeholder constant, or only when the render contract actually changes?
3. Should relay metrics be stored only as append-only raw events at first, or should sampled `/health` snapshots also be persisted for waiting/pairs history?
4. When multi-relay arrives, should aggregation happen by log shipping, shared storage, or a separate admin/metrics service?

## Recommendation

Start with optional registration metadata and relay-side event logging.

That provides the best leverage per unit of risk:

- minimal compatibility exposure
- immediate visibility into real deployed versions
- better confidence for future hosted frontend and protocol changes
- a clean foundation for multi-relay observability later
