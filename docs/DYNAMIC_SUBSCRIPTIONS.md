# Dynamic Subscription Options

## Overview

Phase 7 introduces dynamic subscription options, allowing DMR peers to dynamically configure their talkgroup subscriptions at runtime without requiring server-side configuration changes. This feature is similar to FreeDMR's static talkgroup management.

## How It Works

Peers can specify their desired talkgroup subscriptions by embedding an OPTIONS string in the RPTC Description field during the connection handshake. The server automatically parses these options and routes traffic accordingly.

## OPTIONS Syntax

The OPTIONS string uses a semicolon-separated key=value format:

```
OPTIONS: TS1=3100,3101;TS2=91;AUTO=600
```

### Supported Keys

- **TS1** - Comma-separated list of talkgroup IDs for Timeslot 1
- **TS2** - Comma-separated list of talkgroup IDs for Timeslot 2
- **AUTO** - Auto-static TTL in seconds (0-3600). When set, subscriptions will expire after this duration
- **DROP** - Set to "ALL" to clear all existing subscriptions
- **UNLINK** - Set to "TS1" or "TS2" to clear specific timeslot subscriptions

Keys are case-insensitive.

## Usage Examples

### Basic Static Subscriptions

Add static talkgroups to both timeslots:

```
Description: My Pi-Star Hotspot | OPTIONS: TS1=3100,3101,3102;TS2=91,92
```

This peer will receive traffic for:
- Talkgroups 3100, 3101, 3102 on Timeslot 1
- Talkgroups 91, 92 on Timeslot 2

### Auto-Static with TTL

Set talkgroups that expire after 10 minutes (600 seconds):

```
Description: Mobile Hotspot | OPTIONS: TS1=3100;AUTO=600
```

After 600 seconds of inactivity, the subscription will expire and can be cleaned up.

### Clear and Reset

Clear all existing subscriptions:

```
Description: Hotspot | OPTIONS: DROP=ALL
```

Clear only Timeslot 2 subscriptions:

```
Description: Hotspot | OPTIONS: UNLINK=TS2
```

### Mixed Configuration

Combine multiple options:

```
Description: My Hotspot | OPTIONS: TS1=3100,3101,3102;TS2=91;AUTO=1800;DROP=ALL
```

This will:
1. Clear all existing subscriptions (DROP=ALL)
2. Add TS1: 3100, 3101, 3102
3. Add TS2: 91
4. Set expiry to 30 minutes (1800 seconds)

## Validation and Limits

- **Maximum talkgroups**: 50 per timeslot
- **AUTO range**: 0-3600 seconds (0-60 minutes)
- **Default AUTO**: 600 seconds (10 minutes) if not specified

Invalid options are silently ignored for backward compatibility.

## Backward Compatibility

- If no OPTIONS are provided, behavior is unchanged (uses static bridge configuration only)
- OPTIONS never override ACL decisions - ACLs remain authoritative
- Static bridge rules and dynamic subscriptions work together
- Existing peers without OPTIONS continue to work normally

## Integration with Bridge Rules

Dynamic subscriptions are combined with static bridge rules:

- Static bridge rules (from config) are always evaluated first
- Dynamic peer subscriptions are then added to the routing targets
- Stream deduplication prevents loops
- ACLs are enforced on both static and dynamic rules

## Expiry and Cleanup

When using the AUTO parameter:

- Subscriptions are timestamped when created/updated
- Expired subscriptions can be cleaned up by calling `SubscriptionState.CleanupExpired()`
- Expired subscriptions are automatically filtered out when checking HasTalkgroup()
- The peer can refresh subscriptions by sending a new OPTIONS string

## API Integration (Future)

While the current implementation uses RPTC Description field embedding, future versions may support:

- `GET /api/peers/{peer_id}/subscriptions` - View current subscriptions
- `POST /api/peers/{peer_id}/subscriptions` - Update subscriptions via API
- `DELETE /api/peers/{peer_id}/subscriptions` - Clear subscriptions

## Security Considerations

- OPTIONS are only accepted from authenticated, connected peers
- ACLs are enforced on requested talkgroups
- Invalid or forbidden talkgroups are denied
- Changes are logged for auditing (when logging is enabled)

## Configuration Examples

### Pi-Star Configuration

In your Pi-Star RPTC configuration, add OPTIONS to the Description field:

```
Description: KE0RGS Pi-Star | OPTIONS: TS1=3100,3101;TS2=91;AUTO=600
```

### OpenSpot Configuration

Set the description field to include OPTIONS:

```
Description: OpenSpot-1234 | OPTIONS: TS1=3100;TS2=91
```

### MMDVM Repeater

Configure the repeater's description with OPTIONS:

```
Description: W1ABC Repeater | OPTIONS: TS1=3100,3101,3102;TS2=91,92,93
```

## Troubleshooting

### Subscriptions Not Working

1. Check that OPTIONS syntax is correct (semicolon-separated, no extra spaces)
2. Verify talkgroup IDs are within allowed limits (max 50 per timeslot)
3. Check that ACLs allow the requested talkgroups
4. Ensure the peer is authenticated and connected

### Subscriptions Expiring Too Quickly

- Increase the AUTO value (up to 3600 seconds)
- Re-send the RPTC packet to refresh subscriptions
- Consider using static bridge rules for permanent subscriptions

### OPTIONS Not Parsed

- Ensure the OPTIONS keyword is in uppercase (or lowercase - both work)
- Check that the format is: `OPTIONS: KEY=VALUE;KEY=VALUE`
- Verify the Description field is not truncated

## Technical Details

### Data Structures

```go
type SubscriptionOptions struct {
    TS1      []uint32 // Talkgroups for TS1
    TS2      []uint32 // Talkgroups for TS2
    Auto     int      // TTL in seconds
    DropAll  bool     // Clear all subscriptions
    UnlinkTS uint8    // Clear specific timeslot (1 or 2)
}

type SubscriptionState struct {
    TS1         map[uint32]time.Time // TGID -> expiry
    TS2         map[uint32]time.Time // TGID -> expiry
    AutoTTL     time.Duration        // Auto-static TTL
    LastUpdated time.Time            // Last update timestamp
}
```

### Thread Safety

All subscription operations are thread-safe using mutex locks:
- Read operations use `RLock()`
- Write operations use `Lock()`
- Safe for concurrent access from multiple goroutines

### Router Integration

The router accepts a callback function to check peer subscriptions:

```go
type PeerSubscriptionChecker func(peerID uint32, tgid uint32, timeslot int) bool

router.SetSubscriptionChecker(func(peerID uint32, tgid uint32, timeslot int) bool {
    peer := peerManager.GetPeer(peerID)
    return peer.HasSubscription(tgid, timeslot)
})
```

## References

- [FreeDMR Static Talkgroups](https://www.freedmr.uk/index.php/static-talk-groups-pi-star/)
- [DMR-Nexus PLAN.md - Phase 7](../PLAN.md#phase-7-dynamic-subscription-options-weeks-12-13)
