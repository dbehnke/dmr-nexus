# UI Changes - Before and After

## Active Bridges Section

### Before
```
┌─────────────────────────┐
│ 70777                   │ 👁️
│ 3200449                 │
│ 2 TS2                   │
└─────────────────────────┘
```

### After (When Active)
```
┌─────────────────────────┐
│ 70777                   │ 👁️
│ 3200449 ←─ Link to RadioID.net
│ W7XYZ   ←─ Link to QRZ.com
│ Jane Smith              │
│ Portland, OR, USA       │
│ 2 TS2                   │
└─────────────────────────┘
```

## Recent Transmissions Table

### Before
```
┌──────────┬──────────────┬──────────┬──────────┬──────┐
│ RADIO ID │ TALKGROUP ID │ TIMESLOT │ DURATION │ TIME │
├──────────┼──────────────┼──────────┼──────────┼──────┤
│ 3138617  │ 70777        │ TS2      │ <1s      │ now  │
│ 3200449  │ 70777        │ TS2      │ 32.9s    │ now  │
│ 3200449  │ 70777        │ TS2      │ 9.1s     │ 1m   │
└──────────┴──────────────┴──────────┴──────────┴──────┘
```

### After
```
┌──────────┬──────────┬──────────────┬──────────┬──────────┬──────┐
│ RADIO ID │ CALLSIGN │ TALKGROUP ID │ TIMESLOT │ DURATION │ TIME │
├──────────┼──────────┼──────────────┼──────────┼──────────┼──────┤
│ 3138617* │ K7ABC*   │ 70777        │ TS2      │ <1s      │ now  │
│ 3200449* │ W7XYZ*   │ 70777        │ TS2      │ 32.9s    │ now  │
│ 3200449* │ W7XYZ*   │ 70777        │ TS2      │ 9.1s     │ 1m   │
└──────────┴──────────┴──────────────┴──────────┴──────────┴──────┘
       * = Clickable link
```

## Key Features

### Links Added
1. **Radio ID** → `https://radioid.net/database/view?id={radio_id}`
2. **Callsign** → `https://www.qrz.com/db/{callsign}`

### Information Displayed (Active Bridges)
When someone is actively transmitting on a bridge:
- Radio ID (clickable)
- Callsign (clickable)  
- Full Name (First + Last)
- Location (City, State, Country)

### Information Displayed (Recent Transmissions)
- Radio ID (clickable)
- Callsign (clickable) - NEW COLUMN

### Styling
- Links are styled in blue with hover underline
- Active transmission info shown in red text
- Location info in muted gray
- All links open in new tab with security attributes

## Data Source

The enrichment data comes from:
- **RadioID.net**: https://radioid.net/static/user.csv
- Downloaded on startup and every 24 hours
- Stored in local SQLite database
- ~170,000+ DMR user records

## Graceful Degradation

If user information is not available:
- Radio ID displays without callsign (but still clickable)
- Callsign column shows "-" in table
- No error messages shown to user
