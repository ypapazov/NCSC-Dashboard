# TLP — Traffic Light Protocol

Fresnel implements [TLP v2.0](https://www.first.org/tlp/) to control the visibility of shared information. Every event is assigned a TLP level that determines who can see it.

## TLP levels

| Level | Color | Who can see it |
|-------|-------|----------------|
| **TLP:CLEAR** | White/Grey | No restrictions — anyone with platform access |
| **TLP:GREEN** | Green | All authenticated users on the platform |
| **TLP:AMBER** | Amber | Members of the owning organization, plus organizations with explicit sector access |
| **TLP:AMBER+STRICT** | Amber | Members of the owning organization only |
| **TLP:RED** | Red | Only named recipients (individual users explicitly listed) |

## How TLP is enforced

TLP is enforced at the authorization layer. When you try to view an event, the system checks:

1. Your **role** — some roles (Platform Root, Sector Root, Org Root) bypass TLP entirely
2. Your **organization membership** — determines access for AMBER and AMBER+STRICT
3. Your **user ID** — determines access for RED (named recipients)

You cannot see events above your access level. They simply won't appear in your event list.

## Choosing the right TLP level

- Use **TLP:CLEAR** for information that should be widely available: general advisories, public vulnerabilities, best practice recommendations.
- Use **TLP:GREEN** for information relevant to the whole community but not intended for public release.
- Use **TLP:AMBER** when information should stay within the owning organization and trusted sector partners.
- Use **TLP:AMBER+STRICT** when information must not leave the owning organization under any circumstances.
- Use **TLP:RED** for highly sensitive information that should only reach named individuals — use this sparingly.

## TLP and roles

Administrative roles (Platform Root, Sector Root, Org Root) can see all events regardless of TLP. This is by design — administrators need full visibility to coordinate responses.

Standard roles (Contributor, Viewer, Liaison) are subject to TLP restrictions as described above.
