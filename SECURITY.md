# Security Policy

## Supported versions

Security fixes are provided for the latest stable release and the current release candidate. Older beta builds should be upgraded before reporting compatibility issues.

## Reporting a vulnerability

Use GitHub Security Advisories for this repository to submit vulnerabilities privately. Do not open a public issue containing API keys, administrator tokens, databases, master keys, logs with secrets, or exploit details for an unpatched vulnerability.

Include the affected version, operating system, exposure mode, reproduction steps, and a minimal redacted proof of concept. Reports will be acknowledged after they are reviewed; disclosure timing is coordinated after a fix is available.

The gateway listens on loopback by default. Deployments that enable remote listening without TLS or expose the administration endpoint outside a trusted network are outside the default security boundary.
