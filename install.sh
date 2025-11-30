#!/bin/bash

# Configuration
# Control_plane
CONTROL_PLANE_IP="192.168.1.101"
WORKER_IPS=("192.168.1.102" "192.168.1.103")
SSH_USER="will"
SSH_KEY="~/.ssh/id_ed25519"

# k3s configuration
K3S_VERSION="v1.34.2+k3s1"
CONTROL_PLANE_INSTALL_OPTS="--write-kubeconfig-mode 644 --disable servicelb --disable traefik"
WORKER_INSTALL_OPTS=""

ALL_IPS=("${CONTROL_PLANE_IP}" "${WORKER_IPS[@]}")

WAIT_TIMER=30
ABORT_TIMER=3

SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"

# Function to update locale to fix apt-get warnings
update_locales() {
  echo "=== Configuring locales on all Pis ==="
  for IP in "${ALL_IPS[@]}"; do
    echo -n "Checking locale on ${IP}... "
    ssh ${SSH_OPTS} -i ${SSH_KEY} ${SSH_USER}@${IP} "
        if locale -a | grep -qi 'en_US.utf8'; then
            echo 'already configured' >&2
        else
            echo 'generating...' >&2
            sudo locale-gen en_US.UTF-8 > /dev/null 2>&1
            sudo update-locale LANG=en_US.UTF-8 > /dev/null 2>&1
            echo 'done' >&2
        fi
        " && echo "✓" || echo "✗ Failed"
  done
}

# Fetch and upgrade all standard apt packages, also install btop
update_software() {
  echo "=== Updating all Raspberry Pis ==="
  for IP in "${ALL_IPS[@]}"; do
    echo "Updating ${IP}..."
    ssh ${SSH_OPTS} -i ${SSH_KEY} ${SSH_USER}@${IP} "sudo DEBIAN_FRONTEND=noninteractive apt-get -qq update > /dev/null 2>&1 && sudo DEBIAN_FRONTEND=noninteractive apt-get -qq install -y btop > /dev/null 2>&1 && sudo DEBIAN_FRONTEND=noninteractive apt-get -qq upgrade -y > /dev/null 2>&1" || echo "✗ Failed"

  done
}

# Install k3s server (control plane)
install_control_plane() {
  echo "=== Installing k3s server on p1 (${CONTROL_PLANE_IP}) ==="
  ssh -i ${SSH_OPTS} ${SSH_KEY} ${SSH_USER}@${CONTROL_PLANE_IP} "curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION=${K3S_VERSION} sh -s - server ${CONTROL_PLANE_INSTALL_OPTS}"

  # Wait for server to be ready
  echo "Waiting for k3s server to be ready..."
  sleep ${WAIT_TIMER}

  # Get the server token
  echo "=== Retrieving server token from p1 ==="
  K3S_TOKEN=$(ssh ${SSH_OPTS} -i ${SSH_KEY} ${SSH_USER}@${CONTROL_PLANE_IP} "sudo cat /var/lib/rancher/k3s/server/node-token")

  if [ -z "$K3S_TOKEN" ]; then
    echo "Failed to retrieve k3s token!"
    exit 1
  fi

  echo "Token retrieved successfully"
}

# Install agents on worker nodes
configure_workers() {
  WORKER_NAMES=("p2" "p3")
  for i in "${!WORKER_IPS[@]}"; do
    WORKER_IP="${WORKER_IPS[$i]}"
    WORKER_NAME="${WORKER_NAMES[$i]}"
    echo "=== Installing k3s agent on ${WORKER_NAME} (${WORKER_IP}) ==="
    ssh ${SSH_OPTS} -i ${SSH_KEY} ${SSH_USER}@${WORKER_IP} "curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION=${K3S_VERSION} K3S_URL=https://${CONTROL_PLANE_IP}:6443 K3S_TOKEN=${K3S_TOKEN} sh -s - agent ${WORKER_INSTALL_OPTS}"
    echo "Agent ${WORKER_NAME} installation complete"
  done
}

update_locales
update_software
install_control_plane
configure_workers

echo ""
echo "=== Cluster installation complete ==="
echo "Control plane: p1 (${CONTROL_PLANE_IP})"
echo "Worker nodes: p2 (${WORKER_IPS[0]}), p3 (${WORKER_IPS[1]})"
echo ""
echo "Verify with: ssh ${SSH_USER}@${CONTROL_PLANE_IP} 'sudo kubectl get nodes'"
