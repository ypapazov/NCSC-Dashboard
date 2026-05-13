# Dashboard

The dashboard provides a real-time overview of the cyber security situation across the platform. It is the first thing you see after logging in.

## Hierarchical status tree

The dashboard displays a **hierarchical status tree** organized by:

1. **Sectors** at the top level
2. **Organizations** nested under their respective sectors
3. **Aggregated status** at each level

Each node in the tree shows the assessed status of that sector or organization based on the events and status reports associated with it.

## Assessed statuses

| Status | Color | Meaning |
|--------|-------|---------|
| **Normal** | Green | No active incidents affecting operations |
| **Degraded** | Yellow | Minor issues that don't significantly affect operations |
| **Impaired** | Orange | Notable impact on operations, active response underway |
| **Critical** | Red | Severe impact, major incident in progress |
| **Unknown** | Grey | No recent reports or insufficient data |

## How status is determined

The assessed status for an organization is derived from its most recent **status report**. If no status report has been filed recently, the status shows as Unknown.

Sector-level status is aggregated from the organizations within it — the highest severity among child organizations bubbles up.

## Using the dashboard effectively

- **Expand sectors** to see individual organization statuses.
- **Click on an organization** to see its latest status report and active events.
- Use the dashboard to quickly identify where attention is needed across the platform.
- The dashboard respects your role and TLP visibility — you only see information you are authorized to access.
