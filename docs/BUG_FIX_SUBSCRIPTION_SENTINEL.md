# Bug Fix: Subscription Sentinel Value and Dynamic Bridge Visibility

**Date:** October 21, 2025  
**Issues:** Two bugs affecting dynamic subscription management and dashboard visibility

## Issue 1: Multiple Dynamic Subscriptions on Same Timeslot

### Problem
Peers could be subscribed to multiple dynamic talkgroups on the same timeslot, violating the "one TG per timeslot" design.

### Root Cause
The subscription clearing logic in `AddDynamic()` was checking `!existingExpiry.IsZero()` to identify dynamic subscriptions. However, when `AutoTTL` was 0 (unlimited dynamic subscription), we were storing `time.Time{}` (zero value), which made these subscriptions indistinguishable from static subscriptions (also zero value).

**The bug:**
```go
// Dynamic subscriptions with AutoTTL=0 were stored as:
expiryTime = time.Time{}  // Zero value - looks like static!

// Clearing logic couldn't tell the difference:
if !existingExpiry.IsZero() && existingTGID != tgid {
    delete(tgMap, existingTGID)  // Never deleted unlimited dynamic subs!
}
```

### Solution
Introduced a **sentinel value** to distinguish unlimited dynamic subscriptions from static ones:

- **Static subscriptions** (from RPTC OPTIONS): `time.Time{}` (zero value)
- **TTL-based dynamic subscriptions**: `time.Now().Add(s.AutoTTL)` (future time)
- **Unlimited dynamic subscriptions** (AutoTTL=0): `time.Unix(1, 0)` (January 1, 1970, 00:00:01 UTC)

**The fix in `AddDynamic()`:**
```go
// Add new subscription
var expiryTime time.Time
if s.AutoTTL > 0 {
    expiryTime = time.Now().Add(s.AutoTTL)
} else {
    // Unlimited dynamic subscription - use sentinel value
    expiryTime = time.Unix(1, 0)
}
```

**The fix in clearing logic:**
```go
// Clear all OTHER dynamic subscriptions in this timeslot
// Static subscriptions have time.Time{} (zero value)
// Dynamic subscriptions have sentinel or future time (both non-zero)
for existingTGID, existingExpiry := range tgMap {
    if existingTGID != tgid && !existingExpiry.IsZero() {
        delete(tgMap, existingTGID)  // Now correctly removes unlimited dynamic!
    }
}
```

**Updated `IsSubscribed()` methods:**
```go
// Check if subscription is valid
if expiryTime.IsZero() {
    return true  // Static subscription
}
if expiryTime.Unix() == 1 {
    return true  // Unlimited dynamic (sentinel)
}
return time.Now().Before(expiryTime)  // TTL-based dynamic
```

## Issue 2: Dynamic Bridges Not Showing on Dashboard

### Problem
Dynamic bridges weren't appearing on the dashboard even though subscriptions were working correctly.

### Root Cause
During our refactoring to fix the cross-talk bug, we removed the `GetOrCreateDynamicBridge()` call from the server. We were tracking subscriptions in peer objects but not creating the bridge entries that the dashboard API reads from.

**What was missing:**
The dashboard displays bridges from `router.GetAllDynamicBridges()`, but we weren't creating any `DynamicBridge` objects when peers transmitted on talkgroups.

### Solution
Re-added bridge creation in the server's DMRD handling, but with clear separation of concerns:

```go
// Create/update dynamic bridge for dashboard visibility
// This doesn't affect forwarding logic - it's just for tracking/display
s.router.GetOrCreateDynamicBridge(dmrd.DestinationID, dmrd.Timeslot)
```

**Key architectural decision:**
- **Forwarding logic**: Based ONLY on peer subscription state (`IsSubscribedToTalkgroup()`)
- **Dashboard visibility**: Based on `DynamicBridge` objects in the router
- Bridge objects are **display-only** - they don't control who receives traffic

This maintains our fix for the cross-talk bug while providing dashboard visibility.

## Files Modified

### pkg/peer/subscription.go
- `AddDynamic()` - Uses sentinel value for unlimited dynamic subscriptions
- `ClearAllDynamic()` - Updated comments to reflect sentinel value usage
- `IsSubscribed()` - Handles sentinel value checking
- `IsSubscribedToTalkgroup()` - Handles sentinel value checking

### pkg/network/server.go
- Added `GetOrCreateDynamicBridge()` call after `AddDynamic()` for dashboard visibility
- Maintained subscription-based forwarding logic (no changes to forwarding)

## Testing

After these fixes:

1. ✅ **One TG per timeslot works**: Switching to a new talkgroup clears the previous dynamic subscription
2. ✅ **Unlimited dynamic subscriptions work**: AutoTTL=0 subscriptions persist until manually switched
3. ✅ **TTL-based dynamic subscriptions work**: AutoTTL>0 subscriptions expire after configured time
4. ✅ **Static subscriptions preserved**: TG 4000 clears only dynamic subscriptions
5. ✅ **Dashboard shows dynamic bridges**: Bridge objects created on transmission
6. ✅ **No cross-talk**: Forwarding still based only on peer subscription state

## Subscription Types Summary

| Type | Expiry Time | Created By | Cleared By | Example |
|------|-------------|------------|-----------|---------|
| **Static** | `time.Time{}` (zero) | RPTC OPTIONS | Never (permanent) | Configured TGs |
| **TTL Dynamic** | `time.Now().Add(TTL)` | First transmission | TTL expiry or TG switch | AUTO=600 |
| **Unlimited Dynamic** | `time.Unix(1, 0)` (sentinel) | First transmission | TG switch or TG 4000 | AUTO=0 or no AUTO |

## Benefits of Sentinel Value Approach

1. **Clear distinction** between static and dynamic subscriptions
2. **Backward compatible** with existing static subscription handling
3. **Simple validation** logic in `IsSubscribed()` methods
4. **No performance impact** - just different constant values
5. **Easy to debug** - can visually identify subscription types

## Alternative Approaches Considered

1. **Separate maps for static vs dynamic**: More complex, harder to query
2. **Struct with type flag**: More memory overhead, more complex marshaling
3. **Negative time values**: Confusing, non-standard
4. **Use time.Unix(0, 1) instead of time.Unix(1, 0)**: Both work, chose the more readable one

The sentinel value approach was chosen for its simplicity and clarity.
