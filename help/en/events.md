# Events

Events are the core data objects in Fresnel. Each event represents a cyber security incident, threat, vulnerability, or general observation that needs to be tracked and communicated.

## Creating an event

1. Navigate to **Events** in the sidebar.
2. Click **New Event**.
3. Fill in the required fields:
   - **Title** — A concise description of what happened
   - **Description** — Detailed information (supports Markdown)
   - **Status** — The current state of the event
   - **Impact** — How severe the event is
   - **TLP** — Who should be able to see this event (see [TLP guide](tlp.html))
4. Click **Save** to create the event.

## Event statuses

| Status | Meaning |
|--------|---------|
| **Open** | Newly reported, not yet investigated |
| **Investigating** | Actively being analyzed |
| **Mitigating** | A fix or workaround is being applied |
| **Resolved** | The issue has been addressed |
| **Closed** | No further action needed |

## Impact levels

| Level | Meaning |
|-------|---------|
| **Critical** | Full loss of service or catastrophic data breach |
| **High** | Serious degradation or significant data exposure |
| **Medium** | Partial impact, workarounds available |
| **Low** | Minimal operational impact |
| **Info** | No direct impact, informational only |

## Editing and updating

- Click on any event to view its details.
- Use the **Edit** button to modify fields.
- Use the **Updates** section to add timeline entries without changing the core event data. Each update is timestamped and attributed to the author.

## Attachments

You can attach files to events (IOC lists, screenshots, logs). Navigate to the event detail page and use the **Attachments** section to upload or download files.

## Correlations

Events can be correlated to show relationships — for example, multiple events that share the same threat actor or attack vector. See [Campaigns](campaigns.html) for how to group events at a higher level.

## Revisions

Every change to an event is tracked. Use the **Revisions** tab on the event detail page to see the full edit history.
