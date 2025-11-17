#!/bin/bash
# Video Streaming Test for ShadowMesh P2P Tunnel
# Tests real-world video streaming performance across encrypted tunnel

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}ShadowMesh Video Streaming Test${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if we're running as root (needed for some operations)
if [[ $EUID -eq 0 ]]; then
    SUDO=""
else
    SUDO="sudo"
fi

# Function to print colored messages
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running on server or client
echo ""
echo "Select test mode:"
echo "  1) Server - Host video file (run on 10.0.0.1)"
echo "  2) Client - Stream video (run on 10.0.0.2)"
echo "  3) Generate test video"
echo ""
read -p "Enter choice [1-3]: " MODE

case $MODE in
    1)
        # Server mode - host video file
        print_info "Starting video server mode..."

        # Check if test video exists
        if [ ! -f "test-video.mp4" ]; then
            print_warning "test-video.mp4 not found"
            print_info "Please generate a test video first (option 3) or place a video file named 'test-video.mp4' in this directory"
            echo ""
            echo "Quick option: Download a sample video:"
            echo "  wget https://sample-videos.com/video321/mp4/720/big_buck_bunny_720p_1mb.mp4 -O test-video.mp4"
            exit 1
        fi

        VIDEO_SIZE=$(du -h test-video.mp4 | cut -f1)
        print_info "Found test video: test-video.mp4 (${VIDEO_SIZE})"

        # Get local IP
        LOCAL_IP=$(ip addr show tap0 2>/dev/null | grep "inet " | awk '{print $2}' | cut -d'/' -f1)

        if [ -z "$LOCAL_IP" ]; then
            print_error "TAP device tap0 not found. Is ShadowMesh daemon running?"
            exit 1
        fi

        print_success "TAP device IP: $LOCAL_IP"

        # Start HTTP server
        PORT=8080
        print_info "Starting HTTP server on port $PORT..."
        print_info "Video URL: http://${LOCAL_IP}:${PORT}/test-video.mp4"
        echo ""
        print_success "Server ready! On the client machine, run:"
        echo -e "${GREEN}  curl http://${LOCAL_IP}:${PORT}/test-video.mp4 -o downloaded-video.mp4${NC}"
        echo -e "${GREEN}  # Or with ffplay: ffplay http://${LOCAL_IP}:${PORT}/test-video.mp4${NC}"
        echo -e "${GREEN}  # Or with VLC: vlc http://${LOCAL_IP}:${PORT}/test-video.mp4${NC}"
        echo ""
        print_info "Press Ctrl+C to stop server"
        echo ""

        # Start Python HTTP server
        if command -v python3 &> /dev/null; then
            python3 -m http.server $PORT
        elif command -v python &> /dev/null; then
            python -m SimpleHTTPServer $PORT
        else
            print_error "Python not found. Cannot start HTTP server."
            exit 1
        fi
        ;;

    2)
        # Client mode - stream video
        print_info "Starting video client mode..."

        # Check remote IP
        read -p "Enter server IP address (default: 10.0.0.1): " SERVER_IP
        SERVER_IP=${SERVER_IP:-10.0.0.1}

        read -p "Enter server port (default: 8080): " SERVER_PORT
        SERVER_PORT=${SERVER_PORT:-8080}

        VIDEO_URL="http://${SERVER_IP}:${SERVER_PORT}/test-video.mp4"

        print_info "Testing connectivity to server..."
        if ! ping -c 3 $SERVER_IP > /dev/null 2>&1; then
            print_error "Cannot ping server at $SERVER_IP"
            print_info "Make sure ShadowMesh tunnel is established"
            exit 1
        fi
        print_success "Server is reachable"

        echo ""
        echo "Select streaming method:"
        echo "  1) Download video file (test bandwidth)"
        echo "  2) Stream with ffplay (requires ffmpeg)"
        echo "  3) Stream with VLC"
        echo "  4) Stream with curl + pipe to player"
        echo ""
        read -p "Enter choice [1-4]: " STREAM_METHOD

        case $STREAM_METHOD in
            1)
                print_info "Downloading video from $VIDEO_URL..."
                START_TIME=$(date +%s)

                if curl -o downloaded-video.mp4 --progress-bar $VIDEO_URL; then
                    END_TIME=$(date +%s)
                    DURATION=$((END_TIME - START_TIME))
                    FILE_SIZE=$(du -h downloaded-video.mp4 | cut -f1)

                    print_success "Download complete!"
                    print_info "File size: $FILE_SIZE"
                    print_info "Duration: ${DURATION}s"

                    if [ $DURATION -gt 0 ]; then
                        FILE_SIZE_BYTES=$(stat -f%z downloaded-video.mp4 2>/dev/null || stat -c%s downloaded-video.mp4 2>/dev/null)
                        BANDWIDTH=$((FILE_SIZE_BYTES * 8 / DURATION / 1000000))
                        print_success "Average bandwidth: ${BANDWIDTH} Mbps"
                    fi
                else
                    print_error "Download failed"
                    exit 1
                fi
                ;;

            2)
                if ! command -v ffplay &> /dev/null; then
                    print_error "ffplay not found. Install with: sudo apt install ffmpeg"
                    exit 1
                fi

                print_info "Streaming with ffplay..."
                print_info "Video URL: $VIDEO_URL"
                ffplay -autoexit $VIDEO_URL
                ;;

            3)
                if ! command -v vlc &> /dev/null; then
                    print_error "VLC not found. Install with: sudo apt install vlc"
                    exit 1
                fi

                print_info "Streaming with VLC..."
                print_info "Video URL: $VIDEO_URL"
                vlc $VIDEO_URL
                ;;

            4)
                print_info "Streaming with curl pipe..."

                if command -v mpv &> /dev/null; then
                    curl -s $VIDEO_URL | mpv -
                elif command -v ffplay &> /dev/null; then
                    curl -s $VIDEO_URL | ffplay -
                elif command -v vlc &> /dev/null; then
                    curl -s $VIDEO_URL | vlc -
                else
                    print_error "No video player found (mpv, ffplay, or vlc)"
                    print_info "Install one with: sudo apt install mpv"
                    exit 1
                fi
                ;;

            *)
                print_error "Invalid choice"
                exit 1
                ;;
        esac
        ;;

    3)
        # Generate test video
        print_info "Generating test video..."

        if ! command -v ffmpeg &> /dev/null; then
            print_error "ffmpeg not found. Install with: sudo apt install ffmpeg"
            exit 1
        fi

        echo ""
        echo "Select video quality:"
        echo "  1) Small (480p, ~5MB, 30s)"
        echo "  2) Medium (720p, ~15MB, 30s)"
        echo "  3) Large (1080p, ~30MB, 30s)"
        echo ""
        read -p "Enter choice [1-3]: " QUALITY

        case $QUALITY in
            1)
                RESOLUTION="854x480"
                BITRATE="1M"
                FILENAME="test-video-480p.mp4"
                ;;
            2)
                RESOLUTION="1280x720"
                BITRATE="2.5M"
                FILENAME="test-video-720p.mp4"
                ;;
            3)
                RESOLUTION="1920x1080"
                BITRATE="5M"
                FILENAME="test-video-1080p.mp4"
                ;;
            *)
                print_error "Invalid choice"
                exit 1
                ;;
        esac

        print_info "Generating ${RESOLUTION} test video..."
        print_info "This will create a 30-second video with moving patterns"

        ffmpeg -f lavfi -i testsrc=duration=30:size=${RESOLUTION}:rate=30 \
               -f lavfi -i sine=frequency=1000:duration=30 \
               -c:v libx264 -b:v ${BITRATE} -c:a aac -b:a 128k \
               -y ${FILENAME}

        if [ -f "$FILENAME" ]; then
            ln -sf $FILENAME test-video.mp4
            FILE_SIZE=$(du -h $FILENAME | cut -f1)
            print_success "Test video generated: $FILENAME (${FILE_SIZE})"
            print_info "Symlink created: test-video.mp4 -> $FILENAME"
        else
            print_error "Failed to generate test video"
            exit 1
        fi
        ;;

    *)
        print_error "Invalid choice"
        exit 1
        ;;
esac

print_success "Test complete!"
