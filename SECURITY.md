# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.x     | Best-effort security fixes |

## Reporting a Vulnerability

Email **dios.dev.eight@gmail.com** with:

- Description of the vulnerability
- Steps to reproduce
- Impact assessment
- Suggested fix (optional)

We aim to respond within **72 hours** and will work with you on a coordinated disclosure timeline.

**Please do NOT open public GitHub issues for security vulnerabilities.**

## Scope

In scope:
- Authentication bypass
- Session token leakage
- CSRF protection bypass
- SQL injection in adapter layer
- Cryptographic weaknesses

Out of scope:
- Vulnerabilities in user-supplied configuration (e.g., weak secrets)
- Issues in dependencies (report upstream)
