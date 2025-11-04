# Production Test Quick Start

**Epic 2 Validation**: Ready to test on UK VPS ↔ Belgium Raspberry Pi

---

## 📖 Read the Full Guide on GitHub

**Main Guide**: https://github.com/CG-8663/shadowmesh/blob/main/docs/PRODUCTION_TEST_MANUAL.md

Open this in your browser while you work through the deployment steps.

---

## 🚀 Quick Reference

### 1. Binaries Ready (Local Machine)

Located at:
```
/Users/jamestervit/Webcode/shadowmesh/build/
```

Files:
- `shadowmesh-relay-linux-amd64` (7.0 MB) - For UK VPS
- `shadowmesh-relay-linux-arm64` (6.4 MB) - For Raspberry Pi

### 2. Infrastructure Setup Needed

Before starting, fill in:

```bash
# UK VPS (Proxmox)
UK_VPS_IP="_______________"
UK_VPS_USER="_______________"

# Belgium Raspberry Pi
RPI_IP="_______________"
RPI_USER="pi"
```

### 3. Deployment Steps (Summary)

**Step 1**: Deploy relay server to UK VPS
```bash
cd /Users/jamestervit/Webcode/shadowmesh
scp build/shadowmesh-relay-linux-amd64 $UK_VPS_USER@$UK_VPS_IP:/tmp/shadowmesh-relay
ssh $UK_VPS_USER@$UK_VPS_IP
# Follow guide for setup
```

**Step 2**: Test relay connectivity
```bash
curl http://$UK_VPS_IP:8443/health
```

**Step 3**: Deploy client to Raspberry Pi
```bash
scp build/shadowmesh-relay-linux-arm64 $RPI_USER@$RPI_IP:/tmp/shadowmesh-daemon
ssh $RPI_USER@$RPI_IP
# Follow guide for setup
```

**Step 4**: Test connectivity
```bash
# From Raspberry Pi
ping 10.42.0.3
```

---

## 📋 Validation Checklist

Mark as you complete:

- [ ] Relay server running on UK VPS (port 8443)
- [ ] Relay health endpoint returns 200 OK
- [ ] Client 1 (Raspberry Pi) connected to relay
- [ ] Client 2 (UK VPS) connected to relay
- [ ] Post-quantum handshake complete (4 messages)
- [ ] TAP devices created on both machines
- [ ] Ping works: 10.42.0.2 ↔ 10.42.0.3
- [ ] Frame routing visible in relay logs
- [ ] Heartbeats exchanged every 30s

---

## 📊 Data to Record

During testing, capture:

1. **Handshake Time**: _____ ms
2. **Average Ping Latency**: _____ ms
3. **Throughput** (if tested): _____ Mbps
4. **Any Errors**: _____________________________

---

## 🆘 Quick Troubleshooting

**Relay won't start**:
```bash
sudo ufw allow 8443/tcp
sudo /opt/shadowmesh/shadowmesh-relay --config /etc/shadowmesh/relay.yaml
```

**Client can't connect**:
```bash
curl http://$UK_VPS_IP:8443/health
sudo tail -f /var/log/shadowmesh/relay.log
```

**TAP device fails**:
```bash
sudo modprobe tun
lsmod | grep tun
```

---

## 📖 Full Documentation

- **Complete Manual**: docs/PRODUCTION_TEST_MANUAL.md
- **Epic 2 Summary**: docs/EPIC2_COMPLETION.md
- **Roadmap**: docs/ROADMAP.md

---

## ✅ Success Criteria

Production test is successful when:

✅ Both clients connect to relay
✅ Post-quantum handshake completes
✅ TAP devices functional
✅ Ping works between clients
✅ No connection errors
✅ Performance metrics recorded

---

**Ready to begin testing!** 🎯

Follow the detailed guide at: https://github.com/CG-8663/shadowmesh/blob/main/docs/PRODUCTION_TEST_MANUAL.md
