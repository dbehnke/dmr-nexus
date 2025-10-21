# Dynamic Bridge Subscription Fix

## Issue

Clients were receiving traffic from talkgroups they weren't subscribed to. For example:
- Client 1 transmits on TG 7000
- Client 2 transmits on TG 7001
- Both clients were receiving each other's traffic, even though they weren't subscribed to each other's talkgroups

## Root Cause

The original implementation had two critical flaws:

1. **Automatic Subscriber Addition**: Every peer that transmitted on a talkgroup was automatically added to that talkgroup's dynamic bridge subscriber list
2. **No Subscription Verification**: When forwarding packets, the system didn't verify if peers were actually subscribed to the talkgroup - it just forwarded to everyone in the bridge subscriber list

This meant:
- Client 1 transmits on TG 7000 → Added to TG 7000 bridge
- Client 2 transmits on TG 7001 → Added to TG 7001 bridge  
- Client 1 transmits again on TG 7000 → Forwarded to everyone in bridge (including Client 2)
- Client 2 receives TG 7000 traffic even though they never subscribed to it

## Solution

### 1. Added Subscription Check Method (`pkg/peer/subscription.go`)

```go
// IsSubscribed checks if the peer is subscribed to a specific talkgroup/timeslot
// Returns true if subscribed (either static or dynamic and not expired)
func (s *SubscriptionState) IsSubscribed(tgid uint32, timeslot uint8) bool
```

This method:
- Checks if a talkgroup exists in the peer's subscription map
- Verifies dynamic subscriptions haven't expired (checks TTL)
- Returns true for static subscriptions (zero expiry time)

### 2. Removed Automatic Subscriber Addition

**Before:**
```go
// Add peer to dynamic bridge subscribers for this talkgroup
s.router.AddSubscriberToDynamicBridge(dmrd.DestinationID, dmrd.Timeslot, p.ID)

// Add dynamic subscription to peer (5 minute TTL)
if p.Subscriptions != nil {
    p.Subscriptions.AddDynamic(dmrd.DestinationID, uint8(dmrd.Timeslot), 5*time.Minute)
}
```

**After:**
```go
// Touch the transmitting peer's subscription to this talkgroup
// This extends the TTL if they're already subscribed, or adds them if not
if p.Subscriptions != nil {
    p.Subscriptions.AddDynamic(dmrd.DestinationID, uint8(dmrd.Timeslot), 5*time.Minute)
}
```

We removed the bridge subscriber list management and only track subscriptions at the peer level.

### 3. Implemented Subscription-Based Forwarding (`pkg/network/server.go`)

**New Method: `findDynamicSubscribers()`**
```go
func (s *Server) findDynamicSubscribers(tgid uint32, timeslot uint8, sourcePeerID uint32) []*peer.Peer
```

This method:
- Iterates through all connected peers
- Skips the source peer
- Checks each peer's subscription state using `IsSubscribed()`
- Returns only peers that are actually subscribed to the talkgroup/timeslot

**Updated Forwarding Logic:**
```go
// Forward to dynamically subscribed peers
dynamicTargets := s.findDynamicSubscribers(dmrd.DestinationID, uint8(dmrd.Timeslot), p.ID)

if len(dynamicTargets) > 0 {
    s.forwardToDynamicSubscribers(dmrd, data, dynamicTargets)
}
```

Instead of using a bridge subscriber list, we now query each peer's actual subscription state on every packet.

## How It Works Now

### Scenario 1: Client transmits on subscribed talkgroup
1. Client 1 has static subscription to TG 7000 (from OPTIONS)
2. Client 1 transmits on TG 7000
3. Server calls `AddDynamic()` to extend/refresh the subscription TTL
4. Server calls `findDynamicSubscribers(7000, 2)` 
5. Checks all peers: Is Client 2 subscribed to TG 7000? → No → Skip
6. Only forwards to peers subscribed to TG 7000

### Scenario 2: Client transmits on new talkgroup
1. Client 1 transmits on TG 8000 (never subscribed before)
2. Server calls `AddDynamic(8000, 2, 5min)` → Client 1 now subscribed
3. Server calls `findDynamicSubscribers(8000, 2)`
4. Checks all peers for TG 8000 subscriptions
5. If no other peers subscribed → No forwarding (transmitter talks to themselves)

### Scenario 3: Static + Dynamic subscriptions
1. Client 1 has static subscription: TS1=3100 (from OPTIONS)
2. Client 1 transmits on TG 7000 TS2 → Dynamic subscription created
3. Client 1 now subscribed to: TG 3100 TS1 (static), TG 7000 TS2 (dynamic, 5min TTL)
4. After 5 minutes of inactivity on TG 7000 → Dynamic subscription expires
5. Static subscription to TG 3100 remains forever

## Changes Summary

### Modified Files

1. **pkg/peer/subscription.go**:
   - Added `IsSubscribed(tgid, timeslot)` method
   - Checks both static (zero expiry) and dynamic (TTL-based) subscriptions
   - Thread-safe with RLock

2. **pkg/network/server.go**:
   - Removed `AddSubscriberToDynamicBridge()` call
   - Added `findDynamicSubscribers()` method
   - Modified `forwardToDynamicSubscribers()` to accept peer list instead of IDs
   - Changed routing logic to use subscription-based forwarding

### Benefits

✅ **Accurate forwarding**: Only send to peers that are actually subscribed  
✅ **No cross-contamination**: Peers on different talkgroups don't receive each other's traffic  
✅ **Dynamic subscriptions work correctly**: 5-minute TTL enforced  
✅ **Static subscriptions preserved**: OPTIONS-based subscriptions never expire  
✅ **Proper isolation**: Each talkgroup/timeslot combination is independent  

### Performance Considerations

The new approach queries peer subscriptions on every packet instead of maintaining a cached subscriber list. This is acceptable because:

1. **Small peer count**: Most DMR systems have < 50 peers
2. **Fast check**: `IsSubscribed()` is O(1) hash map lookup
3. **Correct behavior**: Worth the minimal overhead for accurate routing
4. **Cleaner design**: Single source of truth (peer subscriptions)

## Testing

Test the fix by:

1. Connect two clients (Client 1 and Client 2)
2. Client 1 transmits on TG 7000
3. Client 2 transmits on TG 7001
4. **Expected**: Each client only hears their own talkgroup
5. **Before fix**: Both clients heard each other's talkgroups
6. **After fix**: Clients are properly isolated

To verify subscriptions, check the logs:
```
[network.server] [DEBUG] Routing DMRD packet 
    src=3182328 dst=7000 ts=2 
    static_targets=0 dynamic_targets=1
```

The `dynamic_targets` count shows how many peers are actually subscribed.
