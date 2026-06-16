# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.x     | Best-effort security fixes |

## Reporting a Vulnerability

Email **dna.hilman@gmail.com** with:

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

## Reverse proxy & client IP

Goten records a client IP on each session row (`sessions.ip_address`) for audit
and display. It resolves it from `X-Forwarded-For` / `X-Real-IP` /
`CF-Connecting-IP`, taking the **left-most** value, then falling back to the TCP
peer (`RemoteAddr`).

The left-most `X-Forwarded-For` entry is whatever the **client** sent, so it is
only trustworthy if your reverse proxy **overwrites** the header with the real
peer address instead of appending to it:

```nginx
# Safe: the client's own X-Forwarded-For is discarded and replaced.
proxy_set_header X-Forwarded-For $remote_addr;

# UNSAFE for goten: appends, so a client can prepend a forged IP that
# becomes the left-most value goten reads.
# proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
```

Notes:

- This value is **never used as a security control** (no IP allow/deny, no
  rate-limit keying), so spoofing it cannot bypass authentication — the impact is
  limited to misleading audit data.
- If you cannot guarantee an overwriting proxy, set your own trusted header at
  the edge and read it via `goten.GetClientIP(r, "X-Your-Trusted-Header")`.
- Behind multiple proxy hops, configure each hop so the real client IP ends up
  left-most by the time it reaches goten.
