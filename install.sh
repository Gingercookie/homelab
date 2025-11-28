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
COUNTDOWN_TIMER=3

# Function to add all IPs to known_hosts
add_to_known_hosts() {
  echo "=== Adding all IPs to SSH known_hosts ==="
  for IP in "${ALL_IPS[@]}"; do
    echo "Adding ${IP} to known_hosts..."
    ssh-keyscan -H ${IP} >>~/.ssh/known_hosts 2>/dev/null
    if [ $? -eq 0 ]; then
      echo "Successfully added ${IP}"
    else
      echo "Warning: Could not add ${IP} to known_hosts (host may be unreachable)"
    fi
  done
  echo "Known hosts setup complete"
  echo ""
}

# Function to copy SSH key to all machines
copy_ssh_keys() {
  echo "=== Copying SSH keys to all machines ==="
  for IP in "${ALL_IPS[@]}"; do
    echo "Copying SSH key to ${IP}..."
    ssh-copy-id -i ${SSH_KEY} ${SSH_USER}@${IP} 2>/dev/null
    if [ $? -eq 0 ]; then
      echo "Successfully copied key to ${IP}"
    else
      echo "Warning: Could not copy key to ${IP} (may already exist or host unreachable)"
    fi
  done
  echo "SSH key copying complete"
  echo ""
}

# Add all IPs to known_hosts before any SSH operations
add_to_known_hosts

# Copy SSH keys to all machines for inter-node communication
copy_ssh_keys

echo "=== Updating all Raspberry Pis ==="
for IP in "${ALL_IPS[@]}"; do
  echo "Updating ${IP}..."
  ssh -i ${SSH_KEY} ${SSH_USER}@${IP} "sudo apt update && sudo apt upgrade -y"
done

echo "=== Enabling cgroup memory on all Pis ==="
for IP in "${ALL_IPS[@]}"; do
  echo "Configuring cgroup memory on ${IP}..."
  ssh -i ${SSH_KEY} ${SSH_USER}@${IP} "
        if ! grep -q 'cgroup_memory=1 cgroup_enable=memory' /boot/firmware/cmdline.txt; then
            sudo sed -i '\$s/\$/ cgroup_memory=1 cgroup_enable=memory/' /boot/firmware/cmdline.txt
            echo 'cgroup memory enabled - reboot required'
        else
            echo 'cgroup memory already enabled'
        fi
    "
done

echo ""
echo "=== IMPORTANT: Rebooting all Pis for cgroup changes ==="
echo "Press Ctrl+C now if you want to abort, or wait ${ABORT_TIMER} seconds to continue..."
sleep ${ABORT_TIMER}

for IP in "${ALL_IPS[@]}"; do
  echo "Rebooting ${IP}..."
  ssh -i ${SSH_KEY} ${SSH_USER}@${IP} "sudo reboot"
done

echo "Waiting ${WAIT_TIMER} seconds for Pis to reboot..."
sleep ${WAIT_TIMER}

echo "=== Installing k3s server on p1 (${CONTROL_PLANE_IP}) ==="
ssh -i ${SSH_KEY} ${SSH_USER}@${CONTROL_PLANE_IP} "curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION=${K3S_VERSION} sh -s - server ${CONTROL_PLANE_INSTALL_OPTS}"

# Wait for server to be ready
echo "Waiting for k3s server to be ready..."
sleep ${WAIT_TIMER}

# Get the server token
echo "=== Retrieving server token from p1 ==="
K3S_TOKEN=$(ssh -i ${SSH_KEY} ${SSH_USER}@${CONTROL_PLANE_IP} "sudo cat /var/lib/rancher/k3s/server/node-token")

if [ -z "$K3S_TOKEN" ]; then
  echo "Failed to retrieve k3s token!"
  exit 1
fi

echo "Token retrieved successfully"

# Install agents on worker nodes
WORKER_NAMES=("p2" "p3")
for i in "${!WORKER_IPS[@]}"; do
  WORKER_IP="${WORKER_IPS[$i]}"
  WORKER_NAME="${WORKER_NAMES[$i]}"
  echo "=== Installing k3s agent on ${WORKER_NAME} (${WORKER_IP}) ==="
  ssh -i ${SSH_KEY} ${SSH_USER}@${WORKER_IP} "curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION=${K3S_VERSION} K3S_URL=https://${CONTROL_PLANE_IP}:6443 K3S_TOKEN=${K3S_TOKEN} sh -s - agent ${WORKER_INSTALL_OPTS}"
  echo "Agent ${WORKER_NAME} installation complete"
done

echo ""
echo "=== Cluster installation complete ==="
echo "Control plane: p1 (${CONTROL_PLANE_IP})"
echo "Worker nodes: p2 (${WORKER_IPS[0]}), p3 (${WORKER_IPS[1]})"
echo ""
echo "Verify with: ssh ${SSH_USER}@${CONTROL_PLANE_IP} 'sudo kubectl get nodes'"
