# VISION

## PHP Debugging is Hard. We're Making It Simple.

### The Problem

Debugging PHP applications in modern development environments (Docker, WSL2, remote infrastructures) remains unnecessarily complex. While Xdebug provides the necessary capabilities, its integration with containerized workflows creates recurring friction:

- Network configuration between containers and host systems is inconsistent (`host.docker.internal` behavior varies across platforms).
- IDE path mapping requires per-developer setup and is prone to misconfiguration.
- Onboarding new team members involves debugging the debugger before debugging the application.
- Debugging capabilities are rarely available in CI pipelines or staging environments.
- Performance overhead of Xdebug in production-like environments discourages its use.

**The result**: debugging is a local-only activity, not a platform-native capability.

### Our Proposal

**Xdebug-Web** - debugging as a platform service.

We encapsulate the complexity of debugging configuration into a self-contained service. Instead of requiring developers to configure IDEs, manage path mappings, and tune environment variables, they simply run the service and access a web-based debugging interface.

**This is not a debugging tool. This is a debugging layer for your platform.**

### How It Works

1. A lightweight Go service runs alongside your PHP application (typically in the same Docker network).
2. It listens for incoming DBGp connections from Xdebug or PHP Debugger extensions.
3. Upon session establishment, the service manages:
   - Breakpoint registration
   - Step execution (step in/over/out)
   - Call stack retrieval
   - Variable inspection
4. Debugging state is transmitted to a web interface via WebSocket.
5. The developer accesses the interface through a browser, displaying:
   - Source code with current execution point
   - Call stack hierarchy
   - Local, global, and superglobal variables
   - Execution controls (continue, step, stop)

**No IDE configuration. No path mapping. A browser is sufficient.**

### What This Solves

| Problem                                             | Solution                                                |
|-----------------------------------------------------|---------------------------------------------------------|
| Inconsistent Xdebug connectivity in Docker networks | Service operates within the container network           |
| Per-developer IDE configuration inconsistencies     | Unified web interface for all team members              |
| Path mapping discrepancies                          | Shared code volume ensures path alignment               |
| Debugging unavailable in CI/staging                 | Service can be deployed in any environment              |
| Xdebug performance overhead                         | Compatible with lightweight PHP Debugger (~1% overhead) |
| High onboarding friction                            | Intuitive web interface requires no training            |

### Impact on Developer Experience

- **Zero configuration** - developers run `docker-compose up` and access the interface.
- **Consistent environment** - identical interface across all team members, regardless of local setup.
- **Portability** - works across Linux, macOS, Windows, and any environment with Docker support.
- **Immediate feedback** - no context switching between IDE and debugging tools.
- **Transparency** - clear visibility into debugging state and execution flow.

### Impact on Portability

- **IDE-independent** - works with any IDE or no IDE at all.
- **OS-independent** - runs on Linux, macOS, Windows.
- **PHP version-independent** - works with any PHP version supporting Xdebug.
- **Deployment-agnostic** - suitable for local, CI, staging, and (cautiously) production environments.

### The Philosophy

**Xdebug-Web treats debugging as infrastructure, not personal configuration.**

Debugging should be a first-class platform capability-as accessible and consistent as logging, caching, or database access. By decoupling debugging from developer-specific tooling, we eliminate environment drift and reduce cognitive overhead.

We are not replacing IDEs. We are providing an alternative that works where IDE-based debugging introduces friction-particularly in containerized, distributed, or ephemeral environments.

**Debugging should be accessible, consistent, and portable.**

### Roadmap

1. Implement minimal viable product (MVP) with essential debugging functionality.
2. Integrate with PHP Debugger for reduced overhead.
3. Support concurrent debugging sessions.
4. Extend interface capabilities (conditional breakpoints, watches, expression evaluation).
5. Package as Docker image and distribute via GitHub Packages / Docker Hub.
