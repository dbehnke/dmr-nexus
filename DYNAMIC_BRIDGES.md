# Dynamic Bridge Creation in DMR-Nexus

## Overview

DMR-Nexus now supports **dynamic bridge creation** - bridges are automatically created when peers transmit on talkgroups, without requiring pre-configuration. This provides flexibility while still respecting ACL rules.

## How It Works

### Automatic Bridge Creation

When a client keys up on any talkgroup (e.g., TG 7000):

1. **Dynamic Bridge Created**: A dynamic bridge is automatically created for that talkgroup + timeslot combination
2. **Peer Subscribed**: The transmitting peer is added to the bridge's subscriber list
3. **5-Minute TTL**: The peer's subscription is set to expire after 5 minutes of inactivity
4. **Traffic Forwarding**: Voice packets are forwarded to all other peers subscribed to the same talkgroup

### Subscription Types

#### Static Subscriptions
- Configured via RPTC packet OPTIONS field during peer connection
- Format: `TS1=3100,3101;TS2=91` (comma-separated TGIDs per timeslot)
- Never expire (permanent until peer disconnects)

#### Dynamic Subscriptions
- Created automatically when a peer transmits on a talkgroup
- 5-minute TTL from last transmission
- Extend on each transmission (keep-alive)
- Expire when no activity for 5 minutes

### ACL Enforcement

Dynamic bridges respect server ACL rules:

```yaml
systems:
  master-1:
    use_acl: true
    tg1_acl: "PERMIT:3100-3199"  # Only allow TGs 3100-3199 on TS1
    tg2_acl: "PERMIT:91"         # Only allow TG 91 on TS2
```

If a peer tries to transmit on TG 7000 and it's blocked by ACL, the transmission is dropped and no bridge is created.

## Example Scenarios

### Scenario 1: Single Peer Transmitting

1. **Peer A connects** to the server
2. **Peer A keys up** on TG 7000, TS 2
3. **Dynamic bridge created** for TG 7000:TS2
4. **Peer A subscribed** to the bridge (5-minute TTL)
5. No other peers → no forwarding happens (but bridge exists)
6. **After 5 minutes of silence** → bridge expires and is removed

### Scenario 2: Multiple Peers on Same Talkgroup

1. **Peer A transmits** on TG 7000, TS 2
   - Dynamic bridge created
   - Peer A subscribed
2. **Peer B transmits** on TG 7000, TS 2 (within 5 minutes)
   - Same dynamic bridge used
   - Peer B subscribed
   - Peer A's subscription TTL extended
3. **Peer A transmits again**
   - Voice forwarded to Peer B
   - Both subscriptions extended to 5 minutes
4. **Peer B transmits**
   - Voice forwarded to Peer A
   - Both subscriptions extended
5. **After 5 minutes of no traffic** → both subscriptions expire → bridge removed

### Scenario 3: Static + Dynamic Subscriptions

1. **Peer A connects** with OPTIONS: `TS2=7000` (static subscription)
2. **Peer B transmits** on TG 7000, TS 2 (dynamic subscription)
3. **Bridge forwards** Peer B's voice to Peer A
4. **Peer B stops transmitting** for 5 minutes
   - Peer B's dynamic subscription expires
   - Peer A's static subscription remains
   - Bridge stays active (Peer A still subscribed)
5. **Peer C transmits** on TG 7000, TS 2
   - Voice forwarded to Peer A
   - Peer C gets dynamic subscription

### Scenario 4: ACL Blocking

```yaml
systems:
  master-1:
    use_acl: true
    tg2_acl: "DENY:7000"  # Block TG 7000 on TS2
```

1. **Peer A transmits** on TG 7000, TS 2
2. **ACL check fails** → transmission dropped
3. **No bridge created** → TG 7000 is blocked server-side
4. **Debug log**: `Talkgroup denied by TG2_ACL tg=7000`

## Configuration

### Enable Dynamic Bridges

Dynamic bridges are automatically enabled when a router is configured. No special configuration needed:

```yaml
systems:
  master-1:
    mode: MASTER
    enabled: true
    ip: "0.0.0.0"
    port: 62031
    repeat: false  # Set to false to use dynamic bridges instead of simple repeat
    use_acl: true  # Optional: enable ACL filtering
    tg1_acl: "PERMIT:ALL"
    tg2_acl: "PERMIT:ALL"
```

### Static Bridge Rules (Optional)

You can still define static bridge rules for cross-system bridging:

```yaml
bridges:
  PARROT_BRIDGE:
    - system: "master-1"
      tgid: 7000
      timeslot: 2
      active: true
      on: []   # Always active
      off: []
      timeout: 0
    - system: "master-2"
      tgid: 7000
      timeslot: 2
      active: true
      on: []
      off: []
      timeout: 0
```

## Cleanup and Expiration

### Dynamic Bridge Cleanup

The server runs a cleanup loop every 10 seconds:

1. **Check each dynamic bridge**
2. **Count active subscribers**
3. **Check last activity time**
4. **Remove if**: No subscribers AND inactive for 5+ minutes

### Peer Subscription Cleanup

Each peer's subscription state is checked:

1. **Dynamic subscriptions** expire after 5 minutes of no transmissions
2. **Static subscriptions** never expire (until peer disconnects)
3. **Expired subscriptions removed** from peer state

## Logging

### Debug Logs

When a peer transmits, you'll see:

```
[network.server] Dynamic bridge activity peer_id=318232807 tg=7000 ts=2 src=318232807
[network.server] Routing DMRD packet src=318232807 dst=7000 ts=2 static_targets=0 dynamic_subs=1
```

### Info Logs

When bridges are cleaned up:

```
[network.server] Cleaned up inactive dynamic bridges count=3
```

## API Access

### Get Active Dynamic Bridges

The web API will show active dynamic bridges in the `/api/bridges` endpoint:

```json
{
  "bridges": [
    {
      "tgid": 7000,
      "timeslot": 2,
      "created_at": "2025-10-20T23:15:00Z",
      "last_activity": "2025-10-20T23:19:45Z",
      "subscribers": [318232807, 318232808]
    }
  ]
}
```

## Benefits

1. **Zero Configuration**: No need to pre-define every talkgroup
2. **Resource Efficient**: Bridges only exist when actively used
3. **ACL Enforcement**: Server still controls which TGs are allowed
4. **Flexible**: Supports both static and dynamic subscriptions
5. **Auto-Cleanup**: Inactive bridges removed automatically

## Future Enhancements

- **Per-TG TTL**: Different expiration times for different talkgroups
- **Subscriber Limits**: Max subscribers per dynamic bridge
- **Activity Events**: MQTT/WebSocket notifications when bridges created/destroyed
- **Bridge Statistics**: Track usage patterns and popular talkgroups
