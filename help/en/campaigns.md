# Campaigns

Campaigns allow you to group related events that are part of a coordinated threat or ongoing operation. They provide a higher-level view of threat activity that spans multiple individual events.

## What is a campaign?

A campaign represents a coordinated set of cyber security events — for example:

- A phishing campaign targeting multiple organizations in a sector
- A series of DDoS attacks from the same threat actor
- An ongoing APT operation observed across several entities

Campaigns have a **title**, a **description** (Markdown supported), and a list of **linked events**.

## Creating a campaign

1. Navigate to **Campaigns** in the sidebar.
2. Click **New Campaign**.
3. Fill in the title and description.
4. Click **Save**.

You can also create a campaign directly from the events list by selecting multiple events and choosing **Create Campaign from Selection**.

## Linking events

After creating a campaign, you can link events to it:

1. Open the campaign detail page.
2. Use the **Linked Events** section to add events by searching or browsing.
3. Events can be linked or unlinked at any time.

An event can belong to multiple campaigns. This is useful when a single event is relevant to more than one threat narrative.

## Correlations

Beyond campaigns, Fresnel also supports **event correlations** — direct relationships between pairs of events. Correlations can be:

| Type | Meaning |
|------|---------|
| **Manual** | A human analyst linked two events |
| **Suggested** | The system flagged a potential relationship |
| **Confirmed** | A suggested correlation was verified by an analyst |

You can view correlations on the event detail page and visualize them using the **graph view**, which shows the correlation network as an interactive node graph.
