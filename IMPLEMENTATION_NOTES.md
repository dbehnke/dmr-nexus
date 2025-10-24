# DMR User Enrichment Implementation

## Summary
This implementation adds DMR user information enrichment to the dmr-nexus dashboard by:
1. Periodically syncing the RadioID.net user database
2. Storing user information (callsign, name, location) in SQLite
3. Enriching the dashboard UI with clickable links and detailed information

## Backend Changes

### New Database Models
- **DMRUser** (`pkg/database/models.go`): 
  - Stores radio ID, callsign, first/last name, city, state, country
  - Helper methods for `FullName()` and `Location()` formatting

### New Repository
- **DMRUserRepository** (`pkg/database/dmruser_repository.go`):
  - CRUD operations for DMR users
  - Efficient batch upsert for syncing large datasets
  - Query by radio ID or callsign

### RadioID Syncer
- **Syncer** (`pkg/radioid/syncer.go`):
  - Downloads CSV from https://radioid.net/static/user.csv
  - Parses CSV and stores in database
  - Runs on startup and every 24 hours
  - Handles errors gracefully

### API Enhancements
- **User Lookup Endpoint** (`/api/user/:radio_id`):
  - Returns user information for a specific radio ID
  - Used for on-demand lookups

- **Enhanced DTOs**:
  - `TransmissionDTO` now includes `callsign` field
  - `DynamicBridgeDTO` includes active user info (callsign, name, location)

### Integration
- Main application (`cmd/dmr-nexus/main.go`):
  - Initializes user repository
  - Starts RadioID syncer in background
  - Wires user repository to web API

## Frontend Changes

### Active Bridges Enhancement
Enhanced bridge cards to display detailed information when someone is actively transmitting:
- **Radio ID**: Clickable link to RadioID.net
- **Callsign**: Clickable link to QRZ.com
- **Name**: First and last name of the operator
- **Location**: City, State, Country formatted

### Recent Transmissions Table
Added new column and clickable links:
- **Callsign Column**: New column showing operator callsign
- **Radio ID Links**: Click to view on RadioID.net
- **Callsign Links**: Click to view on QRZ.com

## Testing

Comprehensive test coverage includes:

### Unit Tests
1. **DMRUser Model Tests** (`pkg/database/dmruser_test.go`):
   - `TestDMRUser_FullName`: Tests name formatting
   - `TestDMRUser_Location`: Tests location formatting
   - `TestDMRUserRepository_Upsert`: Tests create/update
   - `TestDMRUserRepository_GetByCallsign`: Tests callsign lookup
   - `TestDMRUserRepository_Count`: Tests counting records
   - `TestDMRUserRepository_UpsertBatch`: Tests batch operations

2. **RadioID Syncer Tests** (`pkg/radioid/syncer_test.go`):
   - `TestSyncer_parseCSV`: Tests CSV parsing
   - `TestSyncer_parseCSV_InvalidData`: Tests error handling
   - `TestNewSyncer`: Tests syncer creation
   - `TestSyncer_Start_Cancellation`: Tests graceful shutdown

### Integration
All existing tests continue to pass, confirming no breaking changes.

## URLs Used

1. **RadioID.net Database**: https://radioid.net/static/user.csv
   - CSV format with DMR user database
   - Periodically synced every 24 hours

2. **RadioID.net Lookup**: https://radioid.net/database/view?id={radio_id}
   - View detailed information for a specific radio ID

3. **QRZ.com Lookup**: https://www.qrz.com/db/{callsign}
   - View ham radio callsign information

## Performance Considerations

1. **Database Efficiency**:
   - Batch upserts in groups of 1000 records
   - Indexed on radio_id (primary key) and callsign
   - Typical database size: ~170,000 users

2. **API Performance**:
   - User lookups use indexed queries
   - Optional enrichment (only when data available)
   - No blocking operations in request path

3. **Sync Performance**:
   - Non-blocking background sync
   - Generous timeout (5 minutes) for large file
   - Graceful error handling

## Security

- No credentials required for RadioID.net CSV
- External links open in new tab with `rel="noopener noreferrer"`
- No SQL injection risks (using GORM ORM)
- CodeQL scan passed with 0 alerts

## Future Enhancements

Possible future improvements:
1. Add caching layer for frequent lookups
2. Support incremental updates instead of full sync
3. Add user search/filter functionality
4. Display statistics about database (last sync time, user count)
5. Add UI indicator for database sync status
