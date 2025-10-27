# Private Call Support

## Overview

DMR-Nexus now supports private (unit-to-unit) calls, allowing direct communication between individual radios across the network. This feature tracks where subscribers are located and routes private calls directly to the appropriate peer.

## How It Works

### Subscriber Location Tracking

When any DMRD packet is received from a peer, DMR-Nexus tracks which radio ID (subscriber) is behind which peer:

- **Automatic Tracking**: Updated on every DMRD packet (both group and private calls)
- **TTL-Based Expiration**: Locations expire after 15 minutes of inactivity
- **Clean Disconnect**: Locations are cleared when a peer disconnects

### Private Call Routing

When a private call is received:

1. **Location Lookup**: The system looks up where the destination radio was last seen
2. **Direct Routing**: If found and still fresh, the packet is sent only to that peer
3. **Drop on Unknown**: If the destination is unknown or stale, the call is dropped (logged at debug level)
4. **No Loopback**: Private calls are never sent back to the source peer

### Comparison with Group Calls

| Feature | Group Calls | Private Calls |
|---------|-------------|---------------|
| Routing Method | Dynamic subscriptions, bridges, repeat mode | Direct peer-to-peer based on location |
| Destination | Talkgroup ID | Radio ID |
| ACL Checks | TG1_ACL / TG2_ACL | SUB_ACL (source only) |
| Forwarding | To all subscribed peers | Only to destination peer |
| Bridge Support | Yes | No (bypasses bridge logic) |

## Configuration

Private call routing is controlled by the `private_calls_enabled` configuration option, which can be set globally or per-system.

### Global Configuration

Enable for all systems:

```yaml
global:
  private_calls_enabled: true  # Enable private call routing globally
```

### Per-System Configuration

Enable for specific MASTER systems only:

```yaml
systems:
  MASTER-1:
    mode: MASTER
    private_calls_enabled: true  # Enable for this system
    # ... other configuration

  MASTER-2:
    mode: MASTER
    private_calls_enabled: false  # Disable for this system
    # ... other configuration
```

**Note**: Private calls are disabled by default for backward compatibility.

## Use Cases

### Typical Deployment

In a repeater network with multiple connected peers:

```
Radio A (3120001) → Peer 1 → DMR-Nexus Master ← Peer 2 ← Radio B (3120002)
```

1. Radio A makes a group call to TG 3100 - location tracked
2. Radio B responds on TG 3100 - location tracked  
3. Radio A initiates a private call to Radio B (3120002)
4. DMR-Nexus routes the call directly to Peer 2
5. Only Radio B receives the private call

### Multi-Site Network

In a network with multiple repeater sites:

- Each site's radios are tracked automatically
- Private calls route across sites without manual configuration
- Stale locations are cleaned up automatically
- Works seamlessly with dynamic talkgroup subscriptions

## Security Considerations

### Access Control

- **Source Validation**: The existing `SUB_ACL` is used to validate the source radio ID
- **No Destination ACL**: Currently, there is no ACL for destination radio IDs
- **Consider Adding**: A `PRIVATE_DST_ACL` could be added if needed to restrict which radios can be called

### Privacy

- Private calls bypass all group call logic (bridges, dynamic subscriptions)
- Only the destination peer receives the traffic
- Call metadata is still logged at debug level

### Location Tracking

- Locations are stored in-memory only (not persisted)
- Automatically expire after 15 minutes
- Cleared on peer disconnect
- No historical location data is maintained

## Troubleshooting

### Private Calls Not Working

1. **Check Configuration**: Ensure `private_calls_enabled: true` is set
2. **Verify Locations**: Enable debug logging to see location tracking:
   ```yaml
   logging:
     level: debug
   ```
3. **Check for Stale Locations**: Radios must have transmitted recently (within 15 minutes)
4. **ACL Issues**: Verify source radio passes `SUB_ACL` checks

### Debug Logging

With `level: debug`, you'll see:

```
[DEBUG] Tracking subscriber location radio_id=3120001 peer_id=312001
[DEBUG] Handling private call src=3120001 dst=3120002 ts=1 source_peer=312001
[INFO] Routing private call src=3120001 dst=3120002 ts=1 source_peer=312001 target_peer=312002 target_callsign=PEER2
```

If the destination is unknown:

```
[DEBUG] Private call destination not found dst=3120002 src=3120001
```

### Common Issues

| Problem | Cause | Solution |
|---------|-------|----------|
| Calls not routing | Feature disabled | Set `private_calls_enabled: true` |
| Destination not found | Radio hasn't transmitted recently | Wait for radio to make a group call |
| Stale location | Radio hasn't keyed up in 15+ minutes | Radio needs to transmit to update location |
| ACL denial | Source radio blocked | Check `SUB_ACL` configuration |

## Implementation Details

### Data Structures

- **Location Map**: `map[uint32]*subscriberLocation` - Radio ID → Peer location
- **Location Entry**: Contains peer ID and last seen timestamp
- **Thread Safety**: Protected by `sync.RWMutex` for concurrent access

### Performance

- **Memory**: ~32 bytes per tracked radio (location entry + map overhead)
- **Typical Usage**: For 1000 active radios: ~32 KB memory
- **Cleanup**: Runs every 10 seconds via the existing cleanup loop
- **Lookup**: O(1) hash map lookup for routing decisions

### Edge Cases Handled

1. **Unknown Destination**: Packets are dropped with debug log
2. **Stale Location**: Treated same as unknown (dropped)
3. **Same Peer Source/Dest**: Not forwarded (prevents unnecessary traffic)
4. **Peer Disconnects**: All associated locations are cleared immediately
5. **Race Conditions**: Thread-safe access prevents data races

## Testing

The implementation includes comprehensive tests:

- `TestServer_PrivateCallRouting`: Full end-to-end private call between two peers
- `TestServer_PrivateCallDisabled`: Verifies feature flag behavior
- `TestServer_PrivateCallUnknownDestination`: Unknown destination handling
- `TestServer_SubscriberLocationCleanup`: TTL-based cleanup

All tests pass with the race detector enabled.

## Future Enhancements

Potential improvements for future releases:

1. **Destination ACL**: Add `PRIVATE_DST_ACL` to restrict callable radio IDs
2. **Configurable TTL**: Make the 15-minute timeout configurable
3. **Location Persistence**: Optional database storage for locations across restarts
4. **Fallback Options**: Configurable behavior for unknown destinations (broadcast, drop, NAK)
5. **Statistics**: Track private call counts and success rates
6. **Location Announcements**: Periodic location beacons from peers

## Compatibility

- **Protocol**: Uses standard DMR HomeBrew Protocol (HBP) call type bit (bit 6 of slot byte)
- **Clients**: Compatible with all HBP-compliant clients that support private calls
- **Backward Compatible**: Feature is disabled by default; no impact when disabled
- **hblink3 Comparison**: Provides equivalent or better private call support

## References

- **DMR Protocol**: Call type is encoded in bit 6 of the slot byte (offset 15)
- **HBP Specification**: Standard HomeBrew Protocol for DMR networking
- **Issue Discussion**: See the original issue for implementation requirements and discussion
