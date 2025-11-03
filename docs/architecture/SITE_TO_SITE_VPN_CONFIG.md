<h1>
  <img src="https://pbs.twimg.com/profile_images/1969957304679473152/QW21M-FO_400x400.jpg" alt="Chronara Group Logo" height="60" style="vertical-align: middle; margin-right: 15px;"/>
  Chronara Group ShadowMesh - Site-to-Site Configuration
</h1>

## Secure Encrypted Tunnel: On-Premises to AWS VPC

### Executive Summary
This document provides detailed configuration for establishing a secure IPsec/IKEv2 VPN between on-premises data center and AWS VPC.

---

## Network Planning

### CIDR Ranges

**On-Premises:**
- Internal Network: `192.168.0.0/16`
- VPN Gateway: `192.168.1.1`
- Public IP: `203.0.113.50`

**AWS VPC:**
- VPC CIDR: `10.0.0.0/16`
- Private Subnet: `10.0.1.0/24`
- Public Subnet: `10.0.100.0/24`

---

## IPsec/IKE Security Parameters

### IKEv2 Phase 1 Parameters
```
Encryption: AES-256-GCM
Integrity: SHA-384
DH Group: Group 20 (ECDH P-384)
PRF: HMAC-SHA-384
IKE Lifetime: 28800 seconds (8 hours)
DPD: Enabled (10s interval, 30s timeout)
```

### IPsec Phase 2 Parameters
```
Encryption: AES-256-GCM-128
PFS: Enabled (DH Group 20)
Lifetime: 3600 seconds (1 hour)
Replay Protection: Enabled (window: 128)
```

---

## strongSwan Configuration

### /etc/ipsec.conf
```
config setup
    charondebug="ike 2, knl 2, cfg 2"
    uniqueids=yes

conn aws-tunnel-1
    auto=start
    type=tunnel
    keyexchange=ikev2
    authby=secret
    
    # Local (On-Prem)
    left=%defaultroute
    leftid=203.0.113.50
    leftsubnet=192.168.0.0/16
    
    # Remote (AWS)
    right=198.51.100.10
    rightid=198.51.100.10
    rightsubnet=10.0.0.0/16
    
    # Crypto
    ike=aes256gcm128-sha384-ecp384!
    esp=aes256gcm128-ecp384!
    ikelifetime=28800s
    lifetime=3600s
    
    # DPD
    dpdaction=restart
    dpddelay=10s
    dpdtimeout=30s
```

### /etc/ipsec.secrets
```
203.0.113.50 198.51.100.10 : PSK "YOUR_STRONG_32_CHAR_PSK"
```

---

## AWS Terraform Configuration

```hcl
resource "aws_vpn_gateway" "main" {
  vpc_id = aws_vpc.main.id
  tags = { Name = "shadowmesh-vgw" }
}

resource "aws_customer_gateway" "main" {
  bgp_asn    = 65001
  ip_address = "203.0.113.50"
  type       = "ipsec.1"
  tags = { Name = "shadowmesh-cgw" }
}

resource "aws_vpn_connection" "main" {
  vpn_gateway_id      = aws_vpn_gateway.main.id
  customer_gateway_id = aws_customer_gateway.main.id
  type                = "ipsec.1"
  static_routes_only  = true

  tunnel1_preshared_key                = var.psk_tunnel1
  tunnel1_phase1_encryption_algorithms = ["AES256-GCM-128"]
  tunnel1_phase2_encryption_algorithms = ["AES256-GCM-128"]
  tunnel1_phase1_integrity_algorithms  = ["SHA2-384"]
  tunnel1_phase2_integrity_algorithms  = ["SHA2-384"]
  tunnel1_phase1_dh_group_numbers      = [20]
  tunnel1_phase2_dh_group_numbers      = [20]
  tunnel1_ike_versions                 = ["ikev2"]
}

resource "aws_vpn_connection_route" "office" {
  destination_cidr_block = "192.168.0.0/16"
  vpn_connection_id      = aws_vpn_connection.main.id
}
```

---

## Verification Commands

### strongSwan (On-Premises)
```bash
# Check tunnel status
sudo ipsec status

# Test connectivity
ping 10.0.1.10

# View SA details
sudo ip xfrm state

# Monitor logs
sudo journalctl -u strongswan-starter -f
```

### AWS CLI
```bash
# Check VPN status
aws ec2 describe-vpn-connections \
    --vpn-connection-ids vpn-XXXXX \
    --query 'VpnConnections[0].VgwTelemetry[*].[OutsideIpAddress,Status]'

# Expected output: "UP" for both tunnels
```

### Traffic Flow Test
```bash
# ICMP test
ping -c 5 10.0.1.10

# TCP connectivity
telnet 10.0.1.10 443

# Bandwidth test
iperf3 -c 10.0.1.10 -t 30
```

---

## Troubleshooting

### Tunnel Won't Establish
1. Verify PSK matches on both sides
2. Check crypto parameters alignment
3. Verify firewall allows UDP 500, 4500, ESP
4. Review logs: `sudo journalctl -u strongswan-starter -n 100`

### No Traffic Through Tunnel
1. Check AWS route tables have routes to on-prem CIDR
2. Verify security groups allow traffic from 192.168.0.0/16
3. Confirm IP forwarding enabled: `sysctl net.ipv4.ip_forward`
4. Check iptables FORWARD chain allows VPN traffic

### Performance Issues
1. Test MTU: `ping -M do -s 1400 10.0.1.10`
2. Verify AES-NI support: `grep aes /proc/cpuinfo`
3. Monitor CPU usage during transfers
4. Consider hardware acceleration

---

## Security Recommendations

1. **PSK Management:**
   - Use 32+ character random PSKs
   - Store in AWS Secrets Manager
   - Rotate every 90 days

2. **Monitoring:**
   - Enable VPC Flow Logs
   - Set CloudWatch alarms for tunnel DOWN
   - Monitor failed authentication attempts

3. **Updates:**
   - Keep strongSwan current
   - Apply security patches promptly
   - Test changes in non-production first

4. **Access Control:**
   - Limit SSH access to VPN gateway
   - Use security groups to restrict traffic
   - Implement least privilege principles
