#!/bin/bash

# Creative Tax Docker Compose Runner
# This script manages the docker-compose project
# Domain: creative-tax.local

set -e

DOMAIN="creative-tax.local"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to print colored messages
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    print_error "Docker Compose is not installed or not in PATH"
    exit 1
fi

# Use 'docker compose' if available, otherwise fall back to 'docker-compose'
if docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

# Check if local-secrets.env exists
if [ ! -f "local-secrets.env" ]; then
    print_warn "local-secrets.env not found. Services may fail to start without required environment variables."
fi

# Parse command line arguments
COMMAND=${1:-up}

# Function to check if ports 80/443 are available
check_ports() {
    local port_80_used=$(lsof -i :80 -sTCP:LISTEN -t 2>/dev/null)
    local port_443_used=$(lsof -i :443 -sTCP:LISTEN -t 2>/dev/null)
    
    if [ -n "$port_80_used" ] || [ -n "$port_443_used" ]; then
        print_warn "Ports 80 and/or 443 are in use by another process"
        
        # Check if it's Traefik
        if docker ps --format '{{.Names}}' | grep -q "traefik"; then
            print_warn "Traefik container detected. You may need to stop it first."
            echo -e "${YELLOW}Run this command to stop Traefik:${NC}"
            echo "  docker stop networking-traefik-1"
            echo ""
            read -p "Would you like to stop Traefik now? (y/n) " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                print_info "Stopping Traefik..."
                docker stop networking-traefik-1 2>/dev/null || true
                sleep 2
            else
                print_error "Cannot start application while ports 80/443 are in use"
                return 1
            fi
        else
            print_error "Cannot start application while ports 80/443 are in use"
            return 1
        fi
    fi
    return 0
}

case $COMMAND in
    up)
        if ! check_ports; then
            exit 1
        fi
        print_info "Starting Creative Tax application..."
        $DOCKER_COMPOSE up -d
        print_info "Application started successfully!"
        print_info "Access the application at:"
        print_info "  - http://${DOMAIN}"
        print_info "  - https://${DOMAIN}"
        echo ""
        print_warn "Make sure ${DOMAIN} is in your /etc/hosts file:"
        print_warn "  127.0.0.1 ${DOMAIN}"
        ;;
    down)
        print_info "Stopping Creative Tax application..."
        $DOCKER_COMPOSE down
        print_info "Application stopped successfully!"
        ;;
    restart)
        print_info "Restarting Creative Tax application..."
        $DOCKER_COMPOSE restart
        print_info "Application restarted successfully!"
        ;;
    logs)
        $DOCKER_COMPOSE logs -f
        ;;
    build)
        print_info "Building Creative Tax application..."
        $DOCKER_COMPOSE build
        print_info "Build completed successfully!"
        ;;
    rebuild)
        print_info "Rebuilding and restarting Creative Tax application..."
        $DOCKER_COMPOSE down
        $DOCKER_COMPOSE build
        $DOCKER_COMPOSE up -d
        print_info "Application rebuilt and started successfully!"
        ;;
    status)
        $DOCKER_COMPOSE ps
        ;;
    *)
        echo "Usage: $0 {up|down|restart|logs|build|rebuild|status}"
        echo ""
        echo "Commands:"
        echo "  up       - Start the application (default)"
        echo "  down     - Stop the application"
        echo "  restart  - Restart the application"
        echo "  logs     - View application logs"
        echo "  build    - Build the application images"
        echo "  rebuild  - Rebuild and restart the application"
        echo "  status   - Show service status"
        exit 1
        ;;
esac
