# ShadowMesh UX Expert Handoff Document

**From:** John (Product Manager)
**To:** UX Expert
**Date:** 2025-10-31
**PRD Version:** 0.1
**Status:** Ready for UX Design Phase

---

## Executive Summary

ShadowMesh is a **post-quantum encrypted private network** targeting crypto-native users, enterprise security teams, and privacy-conscious consumers. The MVP focuses on a **Linux CLI client with Grafana monitoring dashboards** - your task is to make this technically complex product accessible and intuitive.

**Your mission:** Design user experiences that enable non-technical users to install, connect, and monitor a quantum-safe private network within 10 minutes.

**Design Philosophy:** "Professional Simplicity" - Enterprise-grade capabilities with zero-configuration simplicity.

---

## Critical Context

### The Challenge

**Technical Complexity vs User Simplicity:**
- Under the hood: ML-KEM-1024 post-quantum crypto, 3-hop onion routing, Ethereum smart contracts, CGNAT traversal
- User experience: "Install, connect, it just works"

**Your Goal:** Hide complexity, surface critical status information, enable troubleshooting without overwhelming users.

### Target Users

**1. Crypto-Native Users (40% of beta target)**
- **Profile:** Comfortable with MetaMask, understand blockchain concepts, run command-line tools
- **Pain Points:** Existing VPNs lack quantum resistance, don't trust centralized relay operators
- **Needs:** Cryptographic proof of security, transparency (public network map), professional monitoring tools
- **Willingness to Pay:** $30-50/month
- **Technical Proficiency:** High (8/10)

**2. Enterprise Security Teams (30% of beta target)**
- **Profile:** IT administrators, security engineers, protect corporate infrastructure
- **Pain Points:** Compliance requires quantum-safe solutions, need audit trails and SLAs
- **Needs:** 99.9% uptime, detailed metrics, troubleshooting capabilities, professional dashboard
- **Willingness to Pay:** $50-200/user/month
- **Technical Proficiency:** High (9/10)

**3. Privacy-Conscious Consumers (30% of beta target)**
- **Profile:** Users in censored countries (China, Iran, Russia), journalists, activists
- **Pain Points:** VPNs blocked by DPI, need censorship resistance, require reliability
- **Needs:** "Just works" experience, clear connection status, simple troubleshooting
- **Willingness to Pay:** $10-20/month
- **Technical Proficiency:** Medium (5/10)

### Success Metrics (MVP)

Your designs must enable:
- **80%+ successful connection rate** within 10 minutes of installation
- **<10 minute installation time** for users without prior blockchain/VPN knowledge
- **At-a-glance status visibility** - user knows "everything is working" or "something is wrong" in <3 seconds
- **100-500 beta user acquisition** via word-of-mouth and community recommendations

---

## Your UX Deliverables

### 1. User Journey Diagrams (3 workflows)

**A. Installation & First-Time Setup Flow**

**Entry Point:** User discovers ShadowMesh via Reddit (r/privacy, r/cryptocurrency) or Product Hunt

**Journey Map:**

| Stage | User Action | System Response | Emotional State | Pain Points |
|-------|-------------|-----------------|-----------------|-------------|
| Discovery | Clicks GitHub repo link | Sees README with "Install in <10 min" | Curious, skeptical | "Is this legit? Will it work on my system?" |
| Download | Copies install command | Terminal shows download progress | Tentative | "Do I need to configure anything?" |
| Installation | Runs `./install.sh` | Script detects Docker, prompts "Install Docker? [Y/n]" | Anxious if Docker missing | "I don't know Docker, will this break my system?" |
| Docker Setup | Presses Y for Docker install | Script installs Docker, shows progress bar | Impatient (waiting) | "How long will this take?" |
| Daemon Install | Waits for completion | Script installs shadowmesh-daemon, systemd service, Docker Compose stack | Passive | "What's it doing behind the scenes?" |
| Passphrase | Prompted for keystore passphrase | System generates hybrid keypair, encrypts keystore | Critical decision moment | "What if I forget the passphrase? Can I recover?" |
| Success | Sees "Dashboard available at http://localhost:8080" | Docker Compose starts Prometheus + Grafana | Relief, excitement | "Is it really working?" |
| Validation | Opens browser to localhost:8080 | Grafana dashboard loads (anonymous auth, no login) | Delight if it works, frustration if fails | "This looks professional!" OR "Port already in use?" |

**Critical UX Questions:**
1. **Passphrase anxiety:** How do we communicate "Write this down, you can't recover it" without scaring users away?
2. **Docker confusion:** If Docker install fails (common on older distros), how do we guide users?
3. **Success confirmation:** How do users know installation succeeded beyond a text message?

**Deliverable Format:** Swimlane diagram with user/system lanes, emotions annotated, decision points highlighted

---

**B. Connection Establishment Flow (Direct P2P vs Relay Fallback)**

**Entry Point:** User runs `shadowmesh connect <peer-id>` in terminal

**Decision Tree:**

```
User runs command
    ↓
Dashboard shows "Connecting..." (yellow, pulsing animation)
    ↓
NAT Type Detection (2-3 seconds)
    ↓
    ├─→ [Direct P2P Path]
    │       ↓
    │   UDP Hole Punching Success (<500ms)
    │       ↓
    │   PQC Handshake (visual progress bar: 0% → 50% → 100%)
    │       ↓
    │   Dashboard shows "Connected - Direct P2P" (GREEN, checkmark icon)
    │       ↓
    │   User sees throughput graph start populating
    │
    └─→ [Relay Fallback Path]
            ↓
        UDP Timeout (Dashboard shows "Direct connection failed, trying relay...")
            ↓
        Query Smart Contract (blockchain icon animation)
            ↓
        Select 3 Relays (World map highlights relay locations)
            ↓
        Establish Multi-Hop Route (progress: Relay1 → Relay2 → Relay3)
            ↓
        Dashboard shows "Connected - Relay-Routed" (YELLOW, relay icon)
            ↓
        Info tooltip: "Your network requires relay routing. Connection may be slower, but still encrypted."

Error Path:
    ↓
All attempts fail (no relays, peer offline, smart contract unreachable)
    ↓
Dashboard shows "Disconnected" (RED, X icon)
    ↓
Error message: "Unable to connect to peer. Possible reasons:
    • Peer is offline
    • No relay nodes available
    • Network connectivity issue

Retrying in 60 seconds... [Retry Now button]"
```

**Critical UX Questions:**
1. **Relay fallback explanation:** How do we explain "CGNAT" and "Symmetric NAT" in plain language?
2. **Wait time anxiety:** 500ms timeout feels long - how do we keep users engaged?
3. **Error recovery:** If connection fails, should we auto-retry or make user click "Retry"?

**Deliverable Format:** Flowchart with decision branches, timing annotations, error handling paths, UI state mockups

---

**C. Dashboard Monitoring Workflow**

**Entry Point:** User opens http://localhost:8080 after connection established

**User Mental Model:** "Is my connection working? How fast is it? Is it secure?"

**Information Hierarchy (Top to Bottom):**

**Row 1: Connection Health (Primary - "Everything OK?" question)**
- At-a-glance status: Large, color-coded "Connected" (green) or "Disconnected" (red)
- Connection type: "Direct P2P" (fast) or "Relay-Routed" (slower but works)
- Peer count: How many peers am I connected to?
- Relay availability: Are relay nodes available if I need them?

**Row 2: Network Performance (Secondary - "How fast is it?" question)**
- Throughput graph: Am I getting good speeds? (Target: 500+ Mbps)
- Latency graph: Is it responsive? (Target: <20ms direct, <70ms relay)
- Packet loss gauge: Is the connection stable? (Target: <1%)

**Row 3: Security Metrics (Tertiary - "Is it really quantum-safe?" question)**
- PQC handshakes: How many secure connections established?
- Key rotation timeline: When did keys last rotate? (proves security is active)
- Crypto CPU usage: How much overhead from encryption?

**Row 4: Peer Map (Context - "Where are my peers?" question)**
- World map: Visual representation of peer locations (city-level markers)
- Relay table: Which relay nodes are available? (sortable by latency)

**User Interaction Patterns:**
1. **Quick Glance (5 seconds):** User looks at Row 1 → Green status → "All good, back to work"
2. **Performance Check (30 seconds):** User sees throughput drop → Hovers over graph → Sees exact Mbps value → Decides if acceptable
3. **Troubleshooting (2-5 minutes):** Connection red → User reads error message → Clicks "Retry Now" → Watches Row 1 for status change
4. **Exploration (first use, 5-10 minutes):** User clicks through panels → Hovers over metrics → Reads tooltips → Understands what each metric means

**Critical UX Questions:**
1. **Information overload:** 4 rows × 10+ panels = too much? How do we prioritize?
2. **Tooltip content:** What explanations do we put in tooltips? E.g., "PQC handshakes: Quantum-safe key exchanges. More = better security."
3. **Mobile responsiveness:** Dashboard designed for desktop (1920x1080), but will users try to view on phone? (Answer: Out of scope for MVP, but flag concern)

**Deliverable Format:** Annotated wireframe showing information hierarchy, interaction points, color coding, tooltip examples

---

### 2. Grafana Dashboard Mockups (3 dashboards)

**A. Main User Dashboard**

**Layout:** 4-row grid, dark theme, chronara.ai logo in top-left header

**Row 1 - Connection Health (4 single-stat panels):**

```
┌─────────────────────┬─────────────────────┬─────────────────────┬─────────────────────┐
│  Connection Status  │  NAT Traversal Type │  Active Peer Count  │  Relay Nodes        │
│                     │                     │                     │  Available          │
│  ● Connected        │  ⚡ Direct P2P       │  1 peer             │  12 nodes           │
│  (green, large)     │  (icon + text)      │  (with sparkline)   │  (last updated 5s)  │
└─────────────────────┴─────────────────────┴─────────────────────┴─────────────────────┘
```

**Row 2 - Network Performance (3 panels):**

```
┌─────────────────────────────────────┬───────────────────────────┬─────────────────────┐
│  Throughput (Mbps)                  │  Latency (ms)             │  Packet Loss (%)    │
│  [Time-series graph]                │  [Time-series graph]      │  [Gauge]            │
│  📈 TX: 520 Mbps (blue line)        │  📊 Avg: 18ms             │  ◉ 0.3%             │
│     RX: 480 Mbps (green line)       │     Range: 15-22ms        │  (green zone)       │
│  Last 1 hour view                   │  Anomaly band visible     │  Threshold: <1%     │
└─────────────────────────────────────┴───────────────────────────┴─────────────────────┘
```

**Row 3 - Security Metrics (3 panels):**

```
┌─────────────────────┬─────────────────────────────────────┬─────────────────────┐
│  PQC Handshakes     │  Key Rotation Timeline              │  Crypto CPU Usage   │
│  3 successful       │  [Event markers on timeline]        │  [Gauge]            │
│  Last 24 hours      │  ● ● ●                              │  ◉ 8%               │
│  (0 failures)       │  2:30pm  5:30pm  8:30pm             │  (green zone)       │
│                     │  Every 5 minutes (as configured)    │  Threshold: <50%    │
└─────────────────────┴─────────────────────────────────────┴─────────────────────┘
```

**Row 4 - Peer Map (2 panels):**

```
┌─────────────────────────────────────┬───────────────────────────────────────────────┐
│  Geographic Distribution            │  Relay Node Table                             │
│  [World map with markers]           │  Node ID    Location       Latency   Uptime   │
│  📍 Peer: San Francisco, USA        │  relay-1    New York, US   12ms      99.8%    │
│  📍 Relay: New York, USA            │  relay-2    London, UK     45ms      99.9%    │
│  📍 Relay: London, UK               │  relay-3    Tokyo, JP      120ms     99.5%    │
│  📍 Relay: Tokyo, Japan             │  (sortable columns)                           │
│  City-level markers, interactive    │  Click row for drill-down (future)            │
└─────────────────────────────────────┴───────────────────────────────────────────────┘
```

**Visual Design Constraints:**
- **Dark theme:** Background #1a1a2e (dark blue-gray), text #eee (light gray)
- **Color palette:**
  - Success green: #00D68F (bright, high contrast)
  - Warning yellow: #FFB300 (amber, noticeable)
  - Danger red: #FF5370 (urgent)
  - Info blue: #4DA6FF (cool, calm)
- **Typography:** Grafana default (Inter font), minimum 12pt for readability at 1920x1080
- **chronara.ai logo:** Top-left header, 40px height, links to chronara.ai homepage
- **Branding footer:** "Powered by chronara.eth" in bottom-right corner

**Interactivity:**
- Hover over time-series graphs → Tooltip shows exact value at timestamp
- Click time-series → Zoom controls appear (1h, 6h, 24h, 7d)
- Relay table rows → Hover highlights, click opens drill-down panel (future feature)

**Accessibility:**
- Color-blind friendly: Use icons + colors (green checkmark, yellow warning triangle, red X)
- Font size: 12pt minimum, scalable with browser zoom (150%, 200% tested)
- High contrast: WCAG AA compliance for text on dark background

**Deliverable Format:** High-fidelity mockup in Figma/Sketch with exact panel layouts, color codes, typography specs, export to JSON for Grafana provisioning

---

**B. Relay Operator Dashboard**

**Purpose:** Enable relay node operators to monitor infrastructure and optimize performance

**Layout:** 3-row grid (less dense than Main User Dashboard)

**Row 1 - Operational Metrics:**
```
┌─────────────────────┬─────────────────────┬─────────────────────┬─────────────────────┐
│  Connected Clients  │  Bandwidth Usage    │  Uptime %           │  Heartbeat Status   │
│  47 clients         │  TX: 1.2 Gbps       │  99.9%              │  ✓ 2 hours ago      │
│  [Time-series 24h]  │  RX: 980 Mbps       │  (Last 7 days)      │  Block #18234567    │
│  Peak: 89 clients   │  [Dual-axis graph]  │  [Single stat]      │  Next: 22h          │
└─────────────────────┴─────────────────────┴─────────────────────┴─────────────────────┘
```

**Row 2 - Economics (Placeholder for Beta):**
```
┌─────────────────────────────────────┬─────────────────────────────────────────────┐
│  Stake Rewards Earned               │  Attestation Status                         │
│  Coming in Beta                     │  Coming in Beta                             │
│  (Placeholder panel)                │  (Placeholder panel)                        │
│  Expected: Token rewards for uptime │  Expected: Proof-of-uptime verification    │
└─────────────────────────────────────┴─────────────────────────────────────────────┘
```

**Row 3 - Client Distribution:**
```
┌───────────────────────────────────────────────────────────────────────────────────┐
│  Geographic Distribution of Connected Clients                                     │
│  [World map with client location markers]                                        │
│  📍 Clients: Clustered in North America (24), Europe (15), Asia (8)              │
│  Privacy-preserving: City-level precision, no IP addresses shown                 │
└───────────────────────────────────────────────────────────────────────────────────┘
```

**Design Consistency:** Reuse Main User Dashboard visual language (same colors, fonts, panel styles)

**Deliverable Format:** Figma/Sketch mockup with operator-specific metrics highlighted

---

**C. Developer/Debug Dashboard**

**Purpose:** Detailed technical metrics for troubleshooting and performance analysis

**Audience:** Power users, developers, support engineers (dense information acceptable)

**Layout:** 5-row grid (high information density)

**Row 1 - Crypto Operation Breakdown:**
```
┌─────────────────────────────────────────────────────────────────────────────────┐
│  Crypto Latency Histogram                                                       │
│  [Histogram showing distribution]                                               │
│  ML-KEM Encapsulation: Avg 15μs, P95 22μs, P99 28μs                            │
│  ML-DSA Signing: Avg 8μs, P95 12μs, P99 15μs                                   │
│  ChaCha20 Encryption: Avg 2μs per frame, P95 3μs, P99 4μs                      │
└─────────────────────────────────────────────────────────────────────────────────┘
```

**Row 2 - Smart Contract Queries:**
```
┌─────────────────────────────────────────────────────────────────────────────────┐
│  Smart Contract Query Latency                                                   │
│  [Time-series with error annotations]                                           │
│  ● Green dots: Successful queries (<200ms)                                      │
│  🔴 Red markers: Failed queries (annotated with error: "Infura rate limit")    │
│  Average: 150ms, Failures: 2 in last 24h                                       │
└─────────────────────────────────────────────────────────────────────────────────┘
```

**Row 3 - Error Logs:**
```
┌─────────────────────────────────────────────────────────────────────────────────┐
│  Error Log Table                                                                │
│  Timestamp         Error Type            Message                    Peer ID     │
│  2025-10-31 14:23  connection_timeout    UDP hole punch failed      peer-abc    │
│  2025-10-31 13:45  smart_contract_error  Infura rate limit exceeded N/A         │
│  2025-10-31 12:10  relay_unavailable     All 3 relays timed out     peer-xyz    │
│  (Scrollable, filterable by error type)                                         │
└─────────────────────────────────────────────────────────────────────────────────┘
```

**Row 4 - System Resources:**
```
┌─────────────┬─────────────┬─────────────┬─────────────────────────────────────┐
│  CPU %      │  Memory MB  │  Disk I/O   │  Network I/O                        │
│  ◉ 12%      │  ◉ 450 MB   │  ◉ 2.1 KB/s │  [Dual-axis graph]                  │
│  (Gauge)    │  (Gauge)    │  (Gauge)    │  TX: 520 Mbps, RX: 480 Mbps         │
└─────────────┴─────────────┴─────────────┴─────────────────────────────────────┘
```

**Row 5 - Pipeline Throughput:**
```
┌─────────────────────────────────────────────────────────────────────────────────┐
│  Frame Encryption Pipeline Throughput (frames/sec)                              │
│  [Bar chart showing stages]                                                     │
│  Capture: 12,500 f/s → Encrypt: 12,480 f/s → Transmit: 12,460 f/s →            │
│  Receive: 12,440 f/s → Decrypt: 12,430 f/s → Inject: 12,420 f/s                │
│  Bottleneck detection: Transmit stage (20 f/s drop)                            │
└─────────────────────────────────────────────────────────────────────────────────┘
```

**Deliverable Format:** Figma/Sketch mockup with annotations explaining each metric's purpose

---

### 3. CLI UX Audit & Recommendations

**A. Help Text Review**

**Current Placeholder:**
```bash
$ shadowmesh --help
ShadowMesh - Post-Quantum Private Network

Commands:
  connect <peer-id>  Connect to a peer
  disconnect         Disconnect from current peer
  status             Show connection status
  logs               View daemon logs
  help               Show this help message

Examples:
  shadowmesh connect peer-abc123
  shadowmesh status
```

**Your Task:**
1. Review for clarity - Is "peer-id" self-explanatory? (No, needs explanation)
2. Add plain-language descriptions - What does each command actually do for the user?
3. Flag jargon - "daemon logs" → "View connection history and troubleshooting info"

**Recommended Revision:**
```bash
$ shadowmesh --help
ShadowMesh - Quantum-Safe Private Network

Commands:
  connect <peer-id>    Establish encrypted connection to another ShadowMesh user
                       Example: shadowmesh connect peer-abc123

  disconnect           End current connection and stop encrypted tunnel

  status               Check if you're connected, connection speed, and security status

  logs                 View connection history and troubleshooting information
                       (Technical details for debugging connection issues)

  help                 Show this help message

Getting Started:
  1. Install: curl https://shadowmesh.network/install.sh | bash
  2. Open dashboard: http://localhost:8080 in your browser
  3. Connect to a peer: shadowmesh connect <peer-id>

Questions? Visit docs.shadowmesh.network or join Discord
```

**Deliverable:** Markdown document with before/after comparisons, rationale for changes

---

**B. Error Message Refinement**

**Bad Error Messages (Cryptic):**
```
❌ "ML-KEM decapsulation failed: invalid ciphertext"
❌ "CGNAT detected"
❌ "Smart contract query timeout"
❌ "TAP device creation failed: permission denied"
```

**Good Error Messages (Actionable):**
```
✅ "Connection failed: Unable to establish secure tunnel with peer.

   Possible reasons:
   • Peer is offline or unreachable
   • Network connectivity issue

   What to do:
   • Check that peer is online: Ask them to run 'shadowmesh status'
   • Retry connection: shadowmesh connect <peer-id>
   • Check dashboard for details: http://localhost:8080

   Still stuck? Visit docs.shadowmesh.network/troubleshooting"
```

```
✅ "Your network requires relay routing to connect.

   Why: Your internet provider uses CGNAT (Carrier-Grade NAT), which blocks direct
   peer-to-peer connections. This is common on cellular and some residential networks.

   What's happening: ShadowMesh is automatically routing your connection through
   relay nodes. Your connection is still encrypted and secure, but may be slightly
   slower.

   Status: Connected via relay (see dashboard for details)"
```

```
✅ "Unable to verify relay nodes on blockchain.

   Reason: Connection to Ethereum network timed out (Infura.io may be experiencing
   issues, or your internet connection is down).

   What to do:
   • Check internet connection: ping google.com
   • Retry in 60 seconds (automatic)
   • Check Infura status: status.infura.io

   Your connection will automatically retry using cached relay node list."
```

```
✅ "Installation requires system permissions.

   Reason: ShadowMesh needs to create a virtual network adapter (TAP device) to
   encrypt your network traffic. This requires administrator permissions.

   What to do:
   • Re-run install script with sudo: sudo ./install.sh
   • This is safe: ShadowMesh is open-source (view code: github.com/shadowmesh)

   Security note: The daemon runs with limited permissions (CAP_NET_ADMIN only),
   not full root access."
```

**Error Message Pattern:**
1. **What happened** (plain language, no jargon)
2. **Why it happened** (optional, if helps user understand)
3. **What to do** (actionable steps, numbered if multiple)
4. **Where to get help** (docs link, Discord, support)

**Deliverable:** Markdown document with error message catalog, before/after comparisons

---

**C. Command Structure Validation**

**Check for Unix Conventions:**
- ✅ Lowercase command names: `shadowmesh connect` (not `shadowMesh Connect`)
- ✅ Hyphen-separated: `shadowmesh connect-peer` (if we had multi-word commands)
- ✅ Consistent verbs: `connect`, `disconnect`, `status` (not `connect`, `stop`, `check-status`)

**Check for User Expectations:**
- ❓ Should `shadowmesh stop` work? (User intuition: "stop" = "disconnect")
  - **Recommendation:** Add `stop` as alias for `disconnect`
- ❓ Should `shadowmesh start` work? (User intuition: "start" = "start daemon")
  - **Recommendation:** Add `start` as alias for daemon startup (systemd does this, but users may try)

**Deliverable:** Markdown document with recommendations for command aliases, consistency improvements

---

### 4. Accessibility Assessment

**Current MVP Scope:** Accessibility intentionally scoped out (target audience: crypto-native users, enterprise IT, command-line users)

**Your Task:** Flag critical accessibility concerns that should be reconsidered

**Assessment Areas:**

**A. Screen Reader Compatibility**
- **Issue:** Grafana's default UI is not optimized for screen readers
- **Impact:** Blind/low-vision users cannot use dashboard effectively
- **Question:** Is this acceptable risk for MVP? (Target audience unlikely to include screen reader users)
- **Recommendation:** Document as known limitation, revisit for Production phase

**B. Keyboard Navigation**
- **Status:** Grafana supports keyboard navigation (Tab, Enter, Arrow keys)
- **Test:** Can user navigate entire dashboard without mouse?
- **Recommendation:** ✅ No changes needed for MVP

**C. Color Blindness**
- **Issue:** Green/red status indicators may be indistinguishable for 8% of male users (deuteranopia)
- **Impact:** User cannot tell "Connected" (green) from "Disconnected" (red) by color alone
- **Recommendation:** ✅ **CRITICAL - Fix in MVP**
  - Add icons: Green checkmark ✓, Red X ✗
  - Add patterns: Green = solid fill, Red = diagonal stripes
  - Add text: "Connected" vs "Disconnected" always visible (not just color)

**D. Font Scaling**
- **Test:** Dashboard readability at browser zoom 150%, 200%
- **Target:** User with low vision can read all text at 200% zoom without horizontal scrolling
- **Recommendation:** Test with Grafana at 150%, 200% zoom, adjust panel sizes if content cuts off

**E. Motion Sensitivity**
- **Issue:** Pulsing "Connecting..." animation may trigger motion sensitivity
- **Recommendation:** Add `prefers-reduced-motion` CSS media query check, disable animations if user prefers

**Deliverable:** 1-2 page accessibility assessment report with:
- Critical issues requiring MVP fix (color blindness)
- Medium priority issues for Beta (screen reader support)
- Low priority issues for Production (advanced keyboard shortcuts)

---

## Design Constraints & Requirements

### Platform Constraints

**Primary Platform:** Web Responsive (Desktop/Laptop)
- **Desktop:** Linux workstations, 1920x1080+ resolution (primary target)
- **Laptop:** 1366x768+ resolution (responsive layout must work)
- **Tablet:** iPad/Android landscape mode (acceptable but not optimized)
- **Mobile:** NOT supported in MVP (Grafana mobile UX is poor, defer to mobile apps in Production)

**Browser Support:**
- Chrome/Edge 90+ (best Grafana performance)
- Firefox 88+ (fully supported)
- Safari 14+ (supported with some limitations on WebGL for map features)

### Visual Design Requirements

**Design System:** Grafana's native dark theme (do not create custom CSS that breaks Grafana upgrades)

**Color Palette (from PRD):**
- Primary: Deep purple/blue #6C63FF (crypto aesthetic)
- Success: Green #00D68F (bright, clear)
- Warning: Amber #FFB300 (noticeable but not alarming)
- Danger: Red #FF5370 (urgent)
- Background: Dark blue-gray #1a1a2e
- Text: Light gray #eeeeee

**Typography:** Grafana default (Inter font family)
- Heading: 18-24pt, semi-bold
- Body: 12-14pt, regular
- Small: 10-11pt, regular (for labels)

**chronara.eth Branding:**
- chronara.ai logo in header (40px height)
- "Powered by chronara.eth" in footer
- Purple accent color (#6C63FF) matches chronara.ai brand

### Information Architecture

**Dashboard Navigation:**
```
Main Navigation (Grafana sidebar)
├── Home → Main User Dashboard (default)
├── Relay Operator → Relay Operator Dashboard
└── Developer → Developer/Debug Dashboard

No custom authentication: Anonymous access for localhost-only dashboards
```

**Critical Path (New User):**
1. Install ShadowMesh
2. Open http://localhost:8080 (bookmark this)
3. See Main User Dashboard by default
4. Glance at Row 1 for status
5. If green → Continue using
6. If red → Read error message in Row 1, follow troubleshooting steps

---

## Open Questions for You

Please document your design decisions on these UX choices:

1. **Getting Started Wizard:** Should we add a first-time setup wizard in Grafana, or assume users read documentation?
   - Pro: Onboarding guidance, higher success rate
   - Con: Adds complexity, users may skip it

2. **Relay Node Selection:** Should relay selection be user-configurable ("Choose relays manually") or always automatic?
   - Current: Automatic based on latency + geographic diversity
   - Power user request: "I want to choose which countries my traffic routes through"

3. **Desktop Notifications:** Should we add browser notifications for connection failures?
   - Pro: Users get alerted even if dashboard not open
   - Con: Notification permission prompt may scare users

4. **Connection Status in Terminal:** Should `shadowmesh status` show rich output (colored, formatted) or plain text (parseable)?
   - Plain text: `status=connected type=direct_p2p latency=18ms`
   - Rich output: `✓ Connected via Direct P2P | 520 Mbps ↑ 480 Mbps ↓ | 18ms latency`

5. **Onboarding Tooltips:** Should we show "?" icons with tooltips for every metric, or expect users to learn over time?
   - More tooltips = less intimidating for new users
   - Fewer tooltips = cleaner UI for experienced users

6. **Error Recovery Automation:** Should system auto-retry failed connections, or require user to click "Retry"?
   - Auto-retry: Convenient, but user may not notice underlying issues
   - Manual retry: User controls when to retry, better for debugging

---

## Timeline & Coordination

**Your Timeline:** 1-2 weeks for UX deliverables

**Parallel Work:**
- Architect: Designing system architecture, database schema (2-3 weeks)
- Epic 1 Development: Monorepo setup and crypto benchmarking can start in parallel

**Dependencies:**
- Architect needs your dashboard mockups for system architecture diagram (show Grafana as component)
- Developers need your CLI error messages for implementation in Epic 2 (Core Networking)

**Handoff Meeting (Recommended):**
- Schedule 1-hour design review with PM and Architect
- Walk through journey diagrams and dashboard mockups
- Get feedback on accessibility decisions
- Align on timeline

---

## Success Criteria for Your Deliverables

**User Journey Diagrams:**
- ✅ All critical user paths documented (happy path + error paths)
- ✅ User emotions and pain points identified
- ✅ Timing annotations show expected duration of each step
- ✅ Decision points clearly marked
- ✅ Diagrams use consistent notation (swimlane format)

**Grafana Dashboard Mockups:**
- ✅ High-fidelity mockups exportable to Grafana JSON
- ✅ All panels have exact specifications (queries, thresholds, colors)
- ✅ Information hierarchy prioritizes at-a-glance status (Row 1 most important)
- ✅ Color-blind accessibility tested (icons + colors, not just colors)
- ✅ Responsive layout tested at 1366x768 (laptop) and 1920x1080 (desktop)

**CLI UX Audit:**
- ✅ All error messages translated to plain language
- ✅ Help text includes examples and next steps
- ✅ Technical jargon explained or replaced
- ✅ Consistent command naming conventions

**Accessibility Assessment:**
- ✅ Critical issues flagged for MVP (color blindness)
- ✅ Medium priority issues documented for Beta (screen reader support)
- ✅ Recommendations actionable (specific fixes, not vague "improve accessibility")

---

## Resources

**Design Tools:**
- Figma (recommended): Free for up to 3 projects
- Sketch: Mac-only, requires license
- Adobe XD: Cross-platform alternative

**Grafana Resources:**
- Grafana Dashboard Examples: https://play.grafana.org/
- Grafana Panel Types: https://grafana.com/docs/grafana/latest/panels-visualizations/
- Grafana Provisioning Docs: https://grafana.com/docs/grafana/latest/administration/provisioning/

**Color Blindness Simulation:**
- Coblis (online): https://www.color-blindness.com/coblis-color-blindness-simulator/
- Stark (Figma plugin): Color contrast checker

**User Research (if time permits):**
- Interview 3-5 crypto-native users about VPN pain points
- Observe installation process (think-aloud protocol)
- Test dashboard comprehension ("What does this metric mean?")

---

## Next Steps

1. **Review this handoff document** - Flag any missing context or unclear requirements
2. **Read full PRD** - Especially UI Design Goals and Requirements sections
3. **Schedule kickoff meeting** - Align with PM and Architect
4. **Begin UX work** - Start with user journey diagrams (highest value)
5. **Weekly check-ins** - 30-min design reviews with PM

**When you're ready to deliver:**
- Create journey diagrams in: `docs/ux/user-journeys.pdf` or Figma link
- Create dashboard mockups in: `docs/ux/dashboard-mockups/` (Figma + JSON exports)
- Create CLI audit in: `docs/ux/cli-ux-audit.md`
- Create accessibility report in: `docs/ux/accessibility-assessment.md`

---

**Questions?** Contact John (PM) via Slack or email.

**Excited to see your designs!** You're creating the UX for the world's first quantum-safe private network. 🎨

---

**Document Version:** 1.0
**Last Updated:** 2025-10-31
**Status:** Ready for UX Expert Review
