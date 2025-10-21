# Talkgroup 4000 - Dynamic Subscription Disconnect

## Overview

Talkgroup 4000 is a special administrative talkgroup that allows peers to disconnect from all their dynamic talkgroup subscriptions while maintaining their static subscriptions.

## Purpose

In DMR-Nexus, peers can dynamically subscribe to talkgroups by transmitting on them. These dynamic subscriptions have a 5-minute TTL (Time To Live) and are automatically renewed when the peer transmits again. However, a peer may want to manually disconnect from all dynamic subscriptions without waiting for them to expire.

## How It Works

When a peer transmits on **TG 4000** (on either timeslot):

1. **Removes peer from all dynamic bridges**: The peer is removed from the subscriber list of all active dynamic bridges
2. **Clears all dynamic subscriptions**: All dynamic talkgroup subscriptions are removed from the peer's subscription state
3. **Preserves static subscriptions**: Static subscriptions configured via the peer's RPTC OPTIONS field remain intact
4. **Logs the action**: An INFO-level log entry is created showing:
   - Peer ID and callsign
   - Number of dynamic bridges disconnected from
   - Number of dynamic subscriptions cleared

## Usage

### From a Radio or Hotspot

1. Select **TG 4000** on your radio
2. Key up briefly (PTT)
3. You will immediately be disconnected from all dynamic talkgroups
4. Your static talkgroups (if configured) remain active

### Example Scenario

**Initial State:**
- Peer KF8S (318232807) connects with static subscription to TG 3100 on TS1
- Peer transmits on TG 7000 TS2 (creates dynamic subscription)
- Peer transmits on TG 8000 TS2 (creates dynamic subscription)
- Current subscriptions: TG 3100 (static), TG 7000 (dynamic), TG 8000 (dynamic)

**After Keying TG 4000:**
- Peer removed from TG 7000 dynamic bridge
- Peer removed from TG 8000 dynamic bridge
- Dynamic subscriptions cleared
- Current subscriptions: TG 3100 (static only)

## Server Logs

When a peer disconnects via TG 4000, you'll see a log entry like:

```
[network.server] [INFO] Peer disconnected from all dynamic talkgroups 
    peer_id=318232807 
    callsign=KF8S 
    dynamic_bridges=2 
    dynamic_subscriptions=2
```

## Technical Details

### Implementation

- **Special TG Check**: The server checks if `DestinationID == 4000` before processing normal bridge logic
- **Router Method**: `RemoveSubscriberFromAllDynamicBridges(peerID)` removes peer from all dynamic bridges
- **Peer Method**: `ClearAllDynamic()` removes dynamic subscriptions while preserving static ones (identified by non-zero expiry times)
- **Early Return**: After processing TG 4000, the function returns without creating a bridge or forwarding packets

### Static vs Dynamic Subscriptions

- **Static Subscriptions**: Configured via RPTC OPTIONS field (e.g., `TS1=3100;TS2=91`), stored with zero expiry time
- **Dynamic Subscriptions**: Created on transmission, stored with 5-minute expiry time
- **Distinction**: The `ClearAllDynamic()` method only removes subscriptions with non-zero expiry times

### ACL Considerations

TG 4000 is processed before ACL checks, so it will work even if TG 4000 is blocked by TG1_ACL or TG2_ACL.

## Configuration

No special configuration is required. TG 4000 is always available as an administrative talkgroup.

### Recommendation

You may want to document TG 4000 for your users and consider adding it to your DMR codeplug as:
- **Name**: "Disconnect Dynamic"
- **TGID**: 4000
- **Timeslot**: 1 or 2 (works on both)
- **Color Code**: Match your repeater/system

## Best Practices

1. **Quick PTT**: A brief key-up is sufficient - no need to hold PTT
2. **Either Timeslot**: Works on TS1 or TS2
3. **No Audio**: Since this is an administrative function, transmitted audio is not forwarded
4. **Immediate Effect**: Disconnection happens instantly, no waiting for TTL expiration

## Compatibility

This feature is unique to DMR-Nexus and is not part of the standard HomeBrew Protocol. Clients/hotspots don't need any special software - they just need to transmit on TG 4000.
