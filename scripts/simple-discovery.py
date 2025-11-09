#!/usr/bin/env python3
"""
Simple HTTP Discovery Server for ShadowMesh v0.1.0-alpha

No database, no Redis - just in-memory peer tracking.
Perfect for getting the network operational quickly.

Usage:
    python3 simple-discovery.py [--port 8080]
"""

from http.server import HTTPServer, BaseHTTPRequestHandler
import json
import time
import argparse
from datetime import datetime

# In-memory peer registry
peers = {}  # {peer_id: {ip, port, is_public, last_seen, registered_at}}

class DiscoveryHandler(BaseHTTPRequestHandler):
    def log_message(self, format, *args):
        """Custom logging with timestamps"""
        timestamp = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        print(f"[{timestamp}] {format % args}")

    def do_POST(self):
        """Handle peer registration"""
        if self.path == '/register' or self.path == '/api/peers/register':
            try:
                content_length = int(self.headers.get('Content-Length', 0))
                if content_length == 0:
                    self.send_error(400, "Empty request body")
                    return

                body = self.rfile.read(content_length)
                data = json.loads(body.decode('utf-8'))

                peer_id = data.get('peer_id')
                ip = data.get('ip') or data.get('ip_address')  # Accept both 'ip' and 'ip_address'
                port = data.get('port', 8443)
                is_public = data.get('is_public', False)

                if not peer_id or not ip:
                    self.send_error(400, "Missing peer_id or ip")
                    return

                now = time.time()
                is_new = peer_id not in peers

                peers[peer_id] = {
                    'peer_id': peer_id,
                    'ip': ip,
                    'port': port,
                    'is_public': is_public,
                    'last_seen': now,
                    'registered_at': peers[peer_id]['registered_at'] if peer_id in peers else now
                }

                # Return 201 Created for successful registration
                self.send_response(201)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                response = {
                    'status': 'ok',
                    'message': 'Peer registered successfully',
                    'peer_id': peer_id
                }
                self.wfile.write(json.dumps(response).encode('utf-8'))

                action = "Registered" if is_new else "Updated"
                self.log_message(f"{action} peer {peer_id[:16]}... at {ip}:{port} (public={is_public})")

            except json.JSONDecodeError:
                self.send_error(400, "Invalid JSON")
            except Exception as e:
                self.send_error(500, str(e))

        elif self.path == '/api/auth/verify':
            # Verify signed challenge - just accept everything for v0.1.0
            try:
                content_length = int(self.headers.get('Content-Length', 0))
                body = self.rfile.read(content_length) if content_length > 0 else b'{}'
                data = json.loads(body.decode('utf-8'))

                peer_id = data.get('peer_id', 'unknown')

                self.send_response(200)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                response = {
                    'status': 'ok',
                    'message': 'Authentication successful',
                    'session_token': f'session-{peer_id}-{int(time.time())}',
                    'authenticated': True
                }
                self.wfile.write(json.dumps(response).encode('utf-8'))

                self.log_message(f"Authenticated peer {peer_id[:16]}...")

            except Exception as e:
                self.send_error(500, str(e))

        elif self.path == '/authenticate' or self.path == '/api/auth/authenticate':
            # Legacy authentication endpoint - redirect to verify
            try:
                content_length = int(self.headers.get('Content-Length', 0))
                body = self.rfile.read(content_length) if content_length > 0 else b'{}'
                data = json.loads(body.decode('utf-8'))

                peer_id = data.get('peer_id', 'unknown')

                self.send_response(200)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                response = {
                    'status': 'ok',
                    'message': 'Authentication successful',
                    'session_token': f'session-{peer_id}-{int(time.time())}'
                }
                self.wfile.write(json.dumps(response).encode('utf-8'))

                self.log_message(f"Authenticated peer {peer_id[:16]}...")

            except Exception as e:
                self.send_error(500, str(e))

        else:
            self.send_error(404, "Not Found")

    def do_GET(self):
        """Handle peer discovery and status endpoints"""
        if self.path.startswith('/api/peers/lookup'):
            try:
                # Parse query parameters
                from urllib.parse import urlparse, parse_qs
                parsed = urlparse(self.path)
                params = parse_qs(parsed.query)

                peer_id = params.get('peer_id', [None])[0]
                count = int(params.get('count', [10])[0])

                if not peer_id:
                    self.send_error(400, "Missing peer_id parameter")
                    return

                # Find matching peers (in simple version, just return the requested peer if it exists)
                now = time.time()
                stale_timeout = 300  # 5 minutes

                matching_peers = []
                if peer_id in peers and (now - peers[peer_id]['last_seen'] < stale_timeout):
                    matching_peers.append({
                        'peer_id': peers[peer_id]['peer_id'],
                        'ip_address': peers[peer_id]['ip'],
                        'port': peers[peer_id]['port'],
                        'is_public': peers[peer_id]['is_public']
                    })

                self.send_response(200)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                response = {
                    'peers': matching_peers,
                    'count': len(matching_peers)
                }
                self.wfile.write(json.dumps(response).encode('utf-8'))

                self.log_message(f"Peer lookup for {peer_id[:16]}... returned {len(matching_peers)} peers")

            except Exception as e:
                self.send_error(500, str(e))

        elif self.path == '/api/auth/challenge':
            # Return a simple challenge for authentication
            try:
                import secrets
                challenge = secrets.token_hex(32)  # 64-char hex string

                self.send_response(200)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                response = {
                    'challenge': challenge,
                    'timestamp': int(time.time())
                }
                self.wfile.write(json.dumps(response).encode('utf-8'))

            except Exception as e:
                self.send_error(500, str(e))

        elif self.path == '/peers' or self.path == '/api/peers':
            try:
                # Remove stale peers (not seen in 5 minutes)
                now = time.time()
                stale_timeout = 300  # 5 minutes
                active_peers = {}

                for peer_id, info in list(peers.items()):
                    if now - info['last_seen'] < stale_timeout:
                        active_peers[peer_id] = info
                    else:
                        self.log_message(f"Removing stale peer {peer_id[:16]}... (last seen {int(now - info['last_seen'])}s ago)")
                        del peers[peer_id]

                self.send_response(200)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                response = {
                    'peers': active_peers,
                    'count': len(active_peers),
                    'timestamp': now
                }
                self.wfile.write(json.dumps(response, indent=2).encode('utf-8'))

                self.log_message(f"Returned {len(active_peers)} active peers")

            except Exception as e:
                self.send_error(500, str(e))

        elif self.path == '/health':
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            response = {
                'status': 'healthy',
                'uptime': int(time.time() - server_start_time),
                'peer_count': len(peers)
            }
            self.wfile.write(json.dumps(response).encode('utf-8'))

        elif self.path == '/stats':
            now = time.time()
            active_peers = [info for info in peers.values() if now - info['last_seen'] < 300]
            public_peers = [p for p in active_peers if p.get('is_public', False)]

            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            response = {
                'total_peers': len(peers),
                'active_peers': len(active_peers),
                'public_peers': len(public_peers),
                'private_peers': len(active_peers) - len(public_peers),
                'uptime_seconds': int(now - server_start_time),
                'server_time': datetime.now().isoformat()
            }
            self.wfile.write(json.dumps(response, indent=2).encode('utf-8'))

        else:
            self.send_error(404, "Not Found")


def run_server(port=8080):
    global server_start_time
    server_start_time = time.time()

    server_address = ('0.0.0.0', port)
    httpd = HTTPServer(server_address, DiscoveryHandler)

    print("=" * 60)
    print("  ShadowMesh Simple Discovery Server v0.1.0")
    print("=" * 60)
    print(f"Listening on: http://0.0.0.0:{port}")
    print(f"Health check: http://0.0.0.0:{port}/health")
    print(f"Stats:        http://0.0.0.0:{port}/stats")
    print(f"Peer list:    http://0.0.0.0:{port}/peers")
    print(f"Started at:   {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print("=" * 60)
    print()

    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        print("\nShutting down gracefully...")
        httpd.server_close()
        print(f"Total peers registered: {len(peers)}")
        print("Shutdown complete")


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='ShadowMesh Simple Discovery Server')
    parser.add_argument('--port', type=int, default=8080, help='HTTP port to listen on (default: 8080)')
    args = parser.parse_args()

    run_server(args.port)
