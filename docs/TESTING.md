# Fresnel — Manual Testing Guide

This document provides step-by-step test procedures for validating the Fresnel application after deployment. Each section covers a functional area with specific test cases, expected results, and troubleshooting tips.

**Prerequisites**: A running Fresnel stack (`make compose-up`). See `docs/DEPLOYMENT.md` for setup.

---

## 0. Infrastructure Health

### 0.1 Health endpoint

```bash
curl -sk https://localhost/api/v1/health | jq .
```

**Expected**: `{"status":"ok","database":"ok","keycloak":"ok"}` (or similar). If `database` or `keycloak` is not `ok`, the corresponding service is unreachable.

### 0.2 Static assets load

Open `https://localhost` in a browser (accept the self-signed cert). The page should show "Authenticating…" briefly before redirecting to Keycloak.

**Check**: View source or DevTools Network tab — `fresnel.css`, `htmx.min.js`, `keycloak.min.js`, `app.js` should all load with HTTP 200.

### 0.3 Keycloak reachable

```bash
curl -s http://localhost:8081/realms/fresnel/.well-known/openid-configuration | jq .issuer
```

**Expected**: `"http://localhost:8081/realms/fresnel"`

---

## 1. Authentication (M1)

### 1.1 Login flow

1. Navigate to `https://localhost/`
2. Keycloak login page appears
3. Enter credentials: `platform-root` / `Fresnel_Test_1!`
4. After login, you should be redirected back to Fresnel
5. The top nav shows the user's name and a "Log out" link
6. The dashboard loads in the main content area

**Troubleshooting**:
- If you see "Missing Keycloak configuration": check that the `<meta>` tags in the HTML source contain the correct Keycloak URL
- If login succeeds but you get a 403 "user not registered": the Keycloak `sub` hasn't been linked yet — check that `009_full_dev_seed.sql` ran and the email matches

### 1.2 Test all 8 users can log in

Log in (or use separate browser profiles / incognito) with each user:

| Username | Password | Expected role shown |
|---|---|---|
| `platform-root` | `Fresnel_Test_1!` | Platform administrator |
| `gov-sector-root` | `Fresnel_Test_1!` | Government sector root |
| `fed-sector-root` | `Fresnel_Test_1!` | Federal sector root |
| `orgA-root` | `Fresnel_Test_1!` | Org A root |
| `orgA-admin` | `Fresnel_Test_1!` | Org A admin |
| `orgA-contributor` | `Fresnel_Test_1!` | Org A contributor |
| `orgA-viewer` | `Fresnel_Test_1!` | Org A viewer |
| `orgB-root` | `Fresnel_Test_1!` | Org B root |

Each user should see the dashboard after login. The first login for each user triggers the email-based `keycloak_sub` linking.

### 1.3 Token refresh

1. Log in as any user
2. Wait 5+ minutes (token refresh happens every 30s, token expires in 10min)
3. Click a navigation link
4. The request should succeed without re-login (token was silently refreshed)

### 1.4 Logout

1. Click "Log out" in the top nav
2. You should be redirected to Keycloak and then back to the Fresnel login page
3. Navigating to `https://localhost/` should require login again

### 1.5 API authentication

```bash
# Get a token (use the Keycloak token endpoint directly)
TOKEN=$(curl -s -X POST "http://localhost:8081/realms/fresnel/protocol/openid-connect/token" \
  -d "client_id=fresnel-app" \
  -d "grant_type=password" \
  -d "username=platform-root" \
  -d "password=Fresnel_Test_1!" | jq -r .access_token)

# Authenticated API call
curl -sk -H "Authorization: Bearer $TOKEN" https://localhost/api/v1/users/me | jq .

# Unauthenticated API call should return 401
curl -sk https://localhost/api/v1/users/me
```

**Expected**: Authenticated call returns user JSON; unauthenticated returns `{"error":"unauthorized"}`.

---

## 2. Authorization (M2)

### 2.1 Platform root sees everything

1. Log in as `platform-root`
2. Navigate to Sectors — should see Government, Finance, Critical Infrastructure
3. Navigate to Organizations — should see all 7 orgs
4. Navigate to Users — should see all 8 users
5. Navigate to Audit — should see audit log entries

### 2.2 Sector root scope restriction

1. Log in as `gov-sector-root`
2. Navigate to Sectors — should see Government and its children (Federal, State)
3. Should NOT see Finance or Critical Infrastructure sectors
4. Navigate to Organizations — should see Org A, Org B, Org C (all in Government hierarchy)
5. Should NOT see Org D–G

### 2.3 Org-scoped user restriction

1. Log in as `orgA-viewer`
2. Navigate to Events — should see only events in Org A (or TLP CLEAR/GREEN from others)
3. Navigate to Organizations — should see Org A
4. Should NOT be able to create events (no create button, or 403 on POST)

### 2.4 Contributor can create events in own org

1. Log in as `orgA-contributor`
2. Navigate to Events → New Event
3. Create an event with title "Test Event", type PHISHING, impact MODERATE, TLP GREEN
4. Event should be created successfully
5. The event's organization should be Org A (the user's active org)

### 2.5 Contributor cannot edit other users' events

1. Log in as `orgA-contributor`
2. Open one of the seeded events created by a different user
3. Try to edit (PUT request) — should get 403 Forbidden

### 2.6 TLP RED visibility

```bash
# As platform-root, create a TLP:RED event with specific recipients
TOKEN_PLATFORM=$(curl -s -X POST "http://localhost:8081/realms/fresnel/protocol/openid-connect/token" \
  -d "client_id=fresnel-app" -d "grant_type=password" \
  -d "username=platform-root" -d "password=Fresnel_Test_1!" | jq -r .access_token)

curl -sk -X POST -H "Authorization: Bearer $TOKEN_PLATFORM" \
  -H "Content-Type: application/json" \
  https://localhost/api/v1/events \
  -d '{
    "title": "TLP RED Test",
    "description": "Secret event",
    "event_type": "DATA_BREACH",
    "tlp": "RED",
    "impact": "CRITICAL",
    "sector_context": "b0000000-0000-4000-8000-000000000002",
    "tlp_red_recipients": ["b1000000-0000-4000-8000-000000000006"]
  }'
```

Then:
- As `orgA-contributor` (who is in the recipients list): should see the event
- As `orgA-viewer` (not in recipients): should NOT see the event

### 2.7 Role assignment

1. Log in as `platform-root`
2. Navigate to Admin → Users
3. Select a user, assign them a new role
4. Verify the role appears in their profile
5. Check audit log — should show the role assignment with HIGH severity

---

## 3. Events (M3)

### 3.1 Create event

1. Log in as `orgA-contributor`
2. Navigate to Events → New Event
3. Fill in:
   - Title: "Phishing Campaign Detected"
   - Description: "Multiple phishing emails targeting staff" (Markdown supported)
   - Event Type: PHISHING
   - TLP: GREEN
   - Impact: MODERATE
   - Sector Context: Federal (should be selectable)
4. Submit
5. **Expected**: Redirected to event detail page. Event has status OPEN.

### 3.2 View event detail

1. Open the created event
2. **Expected**: Title, description (rendered Markdown), metadata (org, sector, submitter, dates), status/impact/TLP badges all display correctly

### 3.3 Create event update

1. On the event detail page, find the "Add Update" section
2. Enter body text: "Identified the phishing domain as example-phish.com"
3. Optionally change impact to HIGH
4. Submit
5. **Expected**: Update appears in the timeline. If impact was changed, the event's impact badge updates.

### 3.4 Status transitions

1. Create an update changing status to INVESTIGATING
2. Create another update changing to MITIGATING
3. Create another changing to RESOLVED
4. **Expected**: Each transition succeeds. The event status badge updates.
5. Try to transition RESOLVED back to OPEN — **Expected**: Validation error (invalid transition)

### 3.5 TLP cannot become less restrictive

1. Create an event with TLP AMBER
2. Try to edit it and change TLP to GREEN
3. **Expected**: Validation error "TLP cannot become less restrictive"

### 3.6 Sector context immutability

1. Create an event in Federal sector
2. Try to edit and change sector_context to a different sector
3. **Expected**: Validation error "sector_context is immutable after creation"

### 3.7 Revision history

1. Edit an event (change title, description)
2. View the revision history section on the detail page
3. **Expected**: Previous version is recorded with the old values and who changed it

### 3.8 Event list filtering

1. Navigate to the events list
2. Filter by status: OPEN
3. Filter by impact: CRITICAL
4. Use the search box
5. **Expected**: List updates dynamically via HTMX

### 3.9 Attachments

*(Only if ClamAV is running and healthy)*

1. On an event detail page, upload a small text file
2. **Expected**: File is scanned and appears in the attachments list
3. Click download — file downloads correctly
4. Try uploading more than 10 files to one event — **Expected**: Error after the 10th

### 3.10 Markdown rendering

1. Create an event with Markdown in the description:
   ```
   ## Heading
   **Bold text** and *italic*
   - List item 1
   - List item 2
   [Link](https://example.com)
   ```
2. **Expected**: Rendered as HTML in the detail view
3. Try XSS vectors in description:
   - `<script>alert('xss')</script>`
   - `<img onerror="alert(1)" src="x">`
   - `[link](javascript:alert(1))`
4. **Expected**: All stripped by bluemonday sanitizer. No alert dialogs.

---

## 4. Status Reports (M4)

### 4.1 Create status report

1. Log in as `orgA-root`
2. Navigate to Status Reports → New
3. Fill in:
   - Title: "Weekly Status - Org A"
   - Scope Type: ORG
   - Scope Ref: Org A
   - Assessed Status: NORMAL
   - Impact: LOW
   - TLP: GREEN
   - Period dates, body text
4. Submit
5. **Expected**: Report created, visible in list

### 4.2 Link events to status report

1. During creation or editing, link 1-2 events
2. View the report detail
3. **Expected**: Linked events appear as clickable links

### 4.3 Revision history

1. Edit the report (change assessed status to DEGRADED)
2. Check revision history
3. **Expected**: Previous version recorded

---

## 5. Campaigns (M4)

### 5.1 Create campaign

1. Log in as `orgA-admin`
2. Navigate to Campaigns → New
3. Title: "Cross-Sector Phishing Wave", TLP: GREEN
4. Submit

### 5.2 Link events to campaign

1. Open the campaign
2. Link 2-3 events (including from different orgs if you have events in multiple orgs)
3. **Expected**: Events appear in the campaign detail

### 5.3 Restricted content indicator

1. Log in as `orgA-viewer`
2. View a campaign that contains events from Org B
3. **Expected**: Org B events show as "restricted content" (the user can't see them, but the count is visible)

---

## 6. Correlations (M4)

### 6.1 Create correlation

```bash
TOKEN=$(curl -s -X POST "http://localhost:8081/realms/fresnel/protocol/openid-connect/token" \
  -d "client_id=fresnel-app" -d "grant_type=password" \
  -d "username=orgA-contributor" -d "password=Fresnel_Test_1!" | jq -r .access_token)

# Get two event IDs from the list
EVENTS=$(curl -sk -H "Authorization: Bearer $TOKEN" https://localhost/api/v1/events | jq -r '.items[0:2][].id')
EVENT_A=$(echo "$EVENTS" | head -1)
EVENT_B=$(echo "$EVENTS" | tail -1)

curl -sk -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  "https://localhost/api/v1/events/$EVENT_A/correlations" \
  -d "{\"event_b_id\": \"$EVENT_B\", \"label\": \"Related phishing domains\"}"
```

**Expected**: Correlation created. Visible on both event detail pages.

### 6.2 Bidirectional visibility

- Correlations should only be visible if the user can see BOTH events

---

## 7. Dashboard (M5)

### 7.1 Hierarchy tree

1. Log in as `platform-root`
2. The dashboard should show the full hierarchy tree:
   - Platform (root)
   - Government → Federal → Org A, Org B
   - Government → State → Org C
   - Finance → Org D, Org E
   - Critical Infrastructure → Energy → Org F
   - Critical Infrastructure → Telecommunications → Org G

### 7.2 Status badges

- Each organization node shows an assessed status based on its latest status report
- Orgs without status reports show UNKNOWN (gray)
- Parent sectors show a computed weighted average of their children

### 7.3 Expand/collapse

- Click on a sector node to expand/collapse its children (HTMX-driven)

### 7.4 Dashboard auto-refresh

- The dashboard should auto-refresh every 60 seconds (check `hx-trigger` attribute)
- Create a status report → within 60s the dashboard should reflect the updated status

### 7.5 Scope restriction

- Log in as `gov-sector-root` — should only see the Government subtree
- Log in as `orgA-viewer` — should see the full tree but restricted nodes show as UNKNOWN

---

## 8. Admin Operations (M2)

### 8.1 Sector CRUD

1. Log in as `platform-root`
2. Navigate to Admin → Sectors
3. Create a new sector "Healthcare" (top-level)
4. Create a subsector "Hospitals" under Healthcare
5. Edit "Hospitals" name to "Hospital Networks"
6. Delete "Hospital Networks"
7. **Expected**: All operations succeed; audit log records each

### 8.2 Organization CRUD

1. Create a new org "City Hospital" in the Healthcare sector
2. Edit its name
3. Delete it
4. **Expected**: Success; audit log entries created

### 8.3 User CRUD

1. Create a new user with email, display name, primary org
2. Edit the user's display name
3. Assign a CONTRIBUTOR role in Org A
4. **Expected**: User appears in list, role assignment recorded in audit

### 8.4 Audit log

1. Navigate to Admin → Audit
2. **Expected**: All mutations from the above tests appear with correct actor, action, resource, severity
3. Filter by severity HIGH — should show role assignments, deletions, root designations

---

## 9. Nudge System (M6)

The nudge system runs on a 15-minute interval. Testing requires either:
- Waiting for the tick cycle
- Or inspecting the logs

### 9.1 Verify scheduler is running

```bash
docker compose -f deploy/docker-compose.yml logs fresnel | grep -i nudge
```

**Expected**: Log entries showing nudge tick activity (may say "0 open events" if none are overdue).

### 9.2 Escalation reset

1. Create a CRITICAL event
2. Create an event update
3. Check that escalation level is reset to 0 (verify via audit log or database query):

```bash
docker compose -f deploy/docker-compose.yml exec postgres \
  psql -U fresnel -c "SELECT * FROM fresnel.escalation_state;"
```

### 9.3 Email delivery (if SMTP configured)

If `SMTP_HOST` is configured:
1. Create a CRITICAL event
2. Wait for the nudge tick (up to 15 minutes)
3. Check the SMTP server for the nudge email

If SMTP is not configured, the logs will show `"email not sent (SMTP not configured)"`.

---

## 10. Federation Stubs (M8)

```bash
TOKEN=$(curl -s -X POST "http://localhost:8081/realms/fresnel/protocol/openid-connect/token" \
  -d "client_id=fresnel-app" -d "grant_type=password" \
  -d "username=platform-root" -d "password=Fresnel_Test_1!" | jq -r .access_token)

curl -sk -H "Authorization: Bearer $TOKEN" https://localhost/api/v1/federation/
```

**Expected**: HTTP 501 with `{"error":"not_implemented","message":"Federation is planned for Phase 2"}`

---

## 11. Formula Stubs (M5)

Custom formulas are deferred. Verify the stubs respond correctly:

```bash
curl -sk -H "Authorization: Bearer $TOKEN" -X PUT \
  -H "Content-Type: application/json" \
  https://localhost/api/v1/sectors/b0000000-0000-4000-8000-000000000001/formula \
  -d '{"source": "def compute(children): return NORMAL"}'
```

**Expected**: Error response indicating custom formulas are not yet available. The dashboard should still compute statuses using the built-in weighted average.

---

## 12. Security Headers

```bash
curl -skI https://localhost/api/v1/health
```

**Expected headers** (check each is present and correct):

| Header | Expected Value |
|---|---|
| `Strict-Transport-Security` | `max-age=63072000; includeSubDomains` |
| `X-Frame-Options` | `DENY` |
| `X-Content-Type-Options` | `nosniff` |
| `Referrer-Policy` | `strict-origin-when-cross-origin` |
| `Content-Security-Policy` | Contains `default-src 'self'` |
| `Permissions-Policy` | Present |

---

## 13. Content Negotiation

### 13.1 JSON response

```bash
curl -sk -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/json" \
  https://localhost/api/v1/events | jq .
```

**Expected**: JSON array of events.

### 13.2 HTML response

```bash
curl -sk -H "Authorization: Bearer $TOKEN" \
  -H "Accept: text/html" \
  https://localhost/api/v1/events
```

**Expected**: HTML fragment (the event list template).

---

## 14. Error Handling

### 14.1 Not found

```bash
curl -sk -H "Authorization: Bearer $TOKEN" \
  https://localhost/api/v1/events/00000000-0000-0000-0000-000000000000
```

**Expected**: HTTP 404

### 14.2 Invalid UUID

```bash
curl -sk -H "Authorization: Bearer $TOKEN" \
  https://localhost/api/v1/events/not-a-uuid
```

**Expected**: HTTP 400

### 14.3 Validation error

```bash
curl -sk -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  https://localhost/api/v1/events \
  -d '{"title": ""}'
```

**Expected**: HTTP 400 with validation error message

---

## 15. Database Integrity

### 15.1 Verify migrations applied

```bash
docker compose -f deploy/docker-compose.yml exec postgres \
  psql -U fresnel -c "SELECT version, filename FROM public.schema_migrations ORDER BY version;"
```

**Expected**: All 9 migrations listed (001 through 009).

### 15.2 Verify seed data

```bash
docker compose -f deploy/docker-compose.yml exec postgres \
  psql -U fresnel -c "SELECT display_name, email FROM fresnel.users ORDER BY display_name;"
```

**Expected**: All 8 test users listed (plus any from earlier migrations 007/008).

### 15.3 Verify IAM records

```bash
docker compose -f deploy/docker-compose.yml exec postgres \
  psql -U fresnel -c "SELECT u.display_name, r.role, r.scope_type FROM fresnel_iam.role_assignments r JOIN fresnel.users u ON u.id = r.user_id ORDER BY u.display_name;"
```

**Expected**: Role assignments for all non-platform-root users.

```bash
docker compose -f deploy/docker-compose.yml exec postgres \
  psql -U fresnel -c "SELECT u.display_name, rd.scope_type FROM fresnel_iam.root_designations rd JOIN fresnel.users u ON u.id = rd.user_id ORDER BY u.display_name;"
```

**Expected**: Root designations for platform-root, gov-sector-root, fed-sector-root, orgA-root, orgB-root.

---

## Quick Smoke Test Script

Run this after a fresh deployment to validate the critical path:

```bash
#!/usr/bin/env bash
set -euo pipefail
BASE="https://localhost"
KC="http://localhost:8081"

echo "=== 1. Health check ==="
curl -sk "$BASE/api/v1/health" | jq .

echo "=== 2. Get token ==="
TOKEN=$(curl -s -X POST "$KC/realms/fresnel/protocol/openid-connect/token" \
  -d "client_id=fresnel-app" -d "grant_type=password" \
  -d "username=platform-root" -d "password=Fresnel_Test_1!" | jq -r .access_token)
echo "Token: ${TOKEN:0:20}..."

echo "=== 3. /users/me ==="
curl -sk -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/users/me" | jq .

echo "=== 4. Dashboard ==="
curl -sk -H "Authorization: Bearer $TOKEN" -H "Accept: application/json" \
  "$BASE/api/v1/dashboard" | jq '.name, .children | length'

echo "=== 5. List events ==="
curl -sk -H "Authorization: Bearer $TOKEN" -H "Accept: application/json" \
  "$BASE/api/v1/events" | jq '.total_count'

echo "=== 6. List sectors ==="
curl -sk -H "Authorization: Bearer $TOKEN" -H "Accept: application/json" \
  "$BASE/api/v1/sectors" | jq 'length'

echo "=== 7. Create event ==="
EVENT_ID=$(curl -sk -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" "$BASE/api/v1/events" \
  -d '{
    "title":"Smoke Test Event","description":"Automated test",
    "event_type":"OTHER","tlp":"GREEN","impact":"INFO",
    "sector_context":"b0000000-0000-4000-8000-000000000002"
  }' | jq -r '.id')
echo "Created event: $EVENT_ID"

echo "=== 8. Get event ==="
curl -sk -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/events/$EVENT_ID" | jq '.title, .status'

echo "=== 9. Federation stub ==="
HTTP_CODE=$(curl -sk -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/federation/")
echo "Federation: HTTP $HTTP_CODE (expect 501)"

echo "=== 10. Security headers ==="
curl -skI "$BASE/api/v1/health" | grep -iE "strict-transport|x-frame|x-content-type|referrer-policy"

echo "=== DONE ==="
```

Save as `scripts/smoke-test.sh` and run after deployment.

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `"user not registered in Fresnel"` (403) | Keycloak user email doesn't match any `fresnel.users` row | Check emails match between `fresnel-realm.json` and `009_full_dev_seed.sql` |
| Template parse error on startup | Missing template function or syntax error in `.html` files | Check `go build` output; all template functions must be in `funcMap()` |
| Dashboard shows all UNKNOWN | No status reports exist, or cache hasn't invalidated | Create a status report; wait up to 60s for cache refresh |
| 502 from nginx | Fresnel container not running or crashing | Check `docker compose logs fresnel` |
| Keycloak redirect loop | Invalid redirect URI config | Check `redirectUris` in `fresnel-realm.json` match your access URL |
| CORS errors in browser console | Web origins mismatch | Check `webOrigins` in `fresnel-realm.json` |
| ClamAV scan failures | ClamAV still loading virus definitions (takes 2-3 min on first start) | Wait for ClamAV healthcheck to pass; check `docker compose logs clamav` |
