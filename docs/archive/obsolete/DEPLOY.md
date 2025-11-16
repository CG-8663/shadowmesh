# 🚀 Deploy ShadowMesh P2P from GitHub

## Quick Deployment (2 Commands)

### 1. Deploy UK VPS (Listener)

```bash
./scripts/deploy-from-github.sh YOUR_VPS_IP YOUR_USERNAME listener
```

Example:
```bash
./scripts/deploy-from-github.sh 123.45.67.89 root listener
```

### 2. Deploy Belgium RPi (Connector)

```bash
./scripts/deploy-from-github.sh YOUR_RPI_IP pi connector YOUR_VPS_IP
```

Example:
```bash
./scripts/deploy-from-github.sh 192.168.1.100 pi connector 123.45.67.89
```

---

## What This Does

The script will automatically:
1. ✅ Test SSH connectivity
2. ✅ Install Git (if needed)
3. ✅ Install Go 1.21.5 (if needed)
4. ✅ Clone/update from GitHub
5. ✅ Build shadowmesh-daemon
6. ✅ Install to `/opt/shadowmesh`
7. ✅ Configure with correct IPs
8. ✅ Verify configuration

**Time**: ~5 minutes per machine

---

## After Deployment

### Start UK VPS (Listener)

```bash
ssh YOUR_USERNAME@YOUR_VPS_IP
sudo /opt/shadowmesh/shadowmesh-daemon --config /etc/shadowmesh/config.yaml
```

You should see:
```
Mode: Listener - Waiting for connections on 0.0.0.0:8443
Listening for P2P connections on 0.0.0.0:8443 (TLS: true)
Daemon running. Press Ctrl+C to stop.
```

### Start Belgium RPi (Connector)

```bash
ssh pi@YOUR_RPI_IP
sudo /opt/shadowmesh/shadowmesh-daemon --config /etc/shadowmesh/config.yaml
```

You should see:
```
Mode: Connector - Connecting to peer: YOUR_VPS_IP:8443
Connected to peer...
Performing post-quantum handshake...
Handshake complete. Session ID: <id>
Tunnel established. Network traffic is now encrypted.
```

### Test the Tunnel

From UK VPS:
```bash
ping 10.10.10.4
```

From Belgium RPi:
```bash
ping 10.10.10.3
```

**Expected**: 0% packet loss ✅

---

## Full Example

```bash
# Local machine - Deploy both
./scripts/deploy-from-github.sh 123.45.67.89 root listener
./scripts/deploy-from-github.sh 192.168.1.100 pi connector 123.45.67.89

# UK VPS - Start listener
ssh root@123.45.67.89
sudo /opt/shadowmesh/shadowmesh-daemon --config /etc/shadowmesh/config.yaml

# Belgium RPi - Start connector (in new terminal)
ssh pi@192.168.1.100
sudo /opt/shadowmesh/shadowmesh-daemon --config /etc/shadowmesh/config.yaml

# Test (from either machine)
ping 10.10.10.4  # From UK VPS
ping 10.10.10.3  # From Belgium RPi
```

---

## Network Configuration

- **UK VPS**: Listener on port 8443, TAP IP 10.10.10.3/24
- **Belgium RPi**: Connector to VPS, TAP IP 10.10.10.4/24
- **Protocol**: WebSocket Secure (WSS)
- **Encryption**: ML-KEM-1024 + ChaCha20-Poly1305

---

## Troubleshooting

### Script fails with "Cannot connect"

Check SSH access:
```bash
ssh YOUR_USERNAME@YOUR_IP
```

### Build fails

The script auto-installs Go, but if it fails:
```bash
ssh YOUR_USERNAME@YOUR_IP
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

### Daemon fails with "Failed to create TAP device"

Run with sudo:
```bash
sudo /opt/shadowmesh/shadowmesh-daemon --config /etc/shadowmesh/config.yaml
```

### Connection refused from connector

Check firewall on UK VPS:
```bash
sudo ufw allow 8443/tcp
# or
sudo iptables -A INPUT -p tcp --dport 8443 -j ACCEPT
```

### Ping doesn't work

Check TAP devices are up:
```bash
ip addr show chr-001
```

If needed, configure manually:
```bash
# UK VPS
sudo ip addr add 10.10.10.3/24 dev chr-001
sudo ip link set chr-001 up

# Belgium RPi
sudo ip addr add 10.10.10.4/24 dev chr-001
sudo ip link set chr-001 up
```

---

## Monitoring

View logs:
```bash
tail -f /var/log/shadowmesh/daemon.log
```

Check statistics (appears every 60 seconds):
```bash
grep "Stats:" /var/log/shadowmesh/daemon.log
```

---

## Re-deployment

To update to latest GitHub code:
```bash
# Just run the script again - it will pull latest
./scripts/deploy-from-github.sh YOUR_IP YOUR_USER listener
./scripts/deploy-from-github.sh YOUR_IP pi connector PEER_IP
```

---

## Architecture

This creates a **direct P2P encrypted tunnel**:

```
UK VPS (Listener)                Belgium RPi (Connector)
┌──────────────────┐             ┌──────────────────┐
│ 10.10.10.3       │◄───────────►│ 10.10.10.4       │
│ chr-001          │   WSS P2P   │ chr-001          │
│                  │   Port 8443 │                  │
│ ML-KEM-1024      │             │ ML-KEM-1024      │
│ ChaCha20-Poly1305│             │ ChaCha20-Poly1305│
└──────────────────┘             └──────────────────┘
```

**No relay needed** - direct encrypted connection!

---

## Next Steps

1. Deploy with commands above
2. Start both daemons
3. Test with ping
4. Run performance tests (see `START_TESTING.md`)
5. Document results
6. Celebrate! 🎉

---

**Questions?** See `START_TESTING.md` for detailed testing guide.
