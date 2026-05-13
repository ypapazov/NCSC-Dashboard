# Getting Started

## Logging in

Fresnel uses single sign-on (SSO) via Keycloak. When you visit the platform for the first time, you'll be redirected to the login page. Enter your credentials or use your organization's identity provider.

After logging in, you'll land on the **Dashboard** — your central overview of the current cyber situation.

## Navigating the platform

The sidebar on the left gives you access to all major sections:

- **Dashboard** — Hierarchical status overview across sectors and organizations
- **Events** — The core of the platform: cyber security incidents, threats, and observations
- **Status Reports** — Periodic reports summarizing the situation for a sector or organization
- **Campaigns** — Groups of related events that form part of a coordinated threat
- **Admin** — User management, roles, sectors, and organizations (visible to administrators)

## Key concepts

### Organizations and sectors

The platform is organized hierarchically:

- **Sectors** represent industry verticals or governmental areas (e.g., Energy, Healthcare, Finance). Sectors can be nested.
- **Organizations** belong to a sector and represent the individual entities that report and consume information.
- Your **primary organization** determines what you see by default and where new events are filed.

### Events

Events are the primary data objects in Fresnel. An event represents a cyber security incident, threat, vulnerability, or observation. Events have:

- A **status** (open, investigating, mitigating, resolved, closed)
- An **impact level** (critical, high, medium, low, info)
- A **TLP level** that controls visibility
- An **owning organization** that created it

### TLP (Traffic Light Protocol)

Every event is tagged with a TLP level that restricts who can see it. See the [TLP guide](tlp.html) for full details.

## Your profile

Click on your name in the sidebar to view your profile. Here you can see your assigned roles, primary organization, and org memberships.
