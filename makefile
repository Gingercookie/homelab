# Configuration
SERVER_IP := 192.168.1.101
AGENT_IPS := 192.168.1.102 192.168.1.103
ALL_IPS := $(SERVER_IP) $(AGENT_IPS)

SSH_USER := will
SSH_KEY := ~/.ssh/id_ed25519
SSH_OPTS := -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null

K3S_VERSION := v1.34.2+k3s1
SERVER_INSTALL_OPTS := --write-kubeconfig-mode 644 --disable servicelb --disable traefik --flannel-backend=none --disable-network-policy
AGENT_INSTALL_OPTS :=

# Wait times (in seconds)
K3S_STARTUP_WAIT := 5

# Colors for output using tput (more portable)
RED := $(shell tput setaf 1 2>/dev/null)
GREEN := $(shell tput setaf 2 2>/dev/null)
YELLOW := $(shell tput setaf 3 2>/dev/null)
BLUE := $(shell tput setaf 4 2>/dev/null)
NC := $(shell tput sgr0 2>/dev/null) # No Color

# Force shell to be bash for color support
SHELL := /bin/bash

.PHONY: all control-plane workers cilium uninstall-all uninstall-control-plane uninstall-workers update-all configure-system help status

## Default target
all: update-all configure-system control-plane cilium workers
	@echo -e "$(GREEN)✓ k3s cluster installation complete!$(NC)"
	@echo -e "$(BLUE)Verify with: make status$(NC)"

## Update and upgrade all nodes
update-all:
	@echo -e "$(BLUE)=== Updating all Raspberry Pis ===$(NC)"
	@for ip in $(ALL_IPS); do \
		echo -n "Updating $$ip... "; \
		ssh $(SSH_OPTS) -i $(SSH_KEY) $(SSH_USER)@$$ip \
			"sudo DEBIAN_FRONTEND=noninteractive apt-get -qq update > /dev/null 2>&1 && \
			 sudo DEBIAN_FRONTEND=noninteractive apt-get -qq upgrade -y > /dev/null 2>&1" \
			&& echo "$(GREEN)✓$(NC)" || echo "$(RED)✗ Failed$(NC)"; \
	done

## Configure system settings (locale)
configure-system:
	@echo -e "$(BLUE)=== Configuring system settings on all Pis ===$(NC)"
	@# Configure locales
	@for ip in $(ALL_IPS); do \
		echo -n "Configuring locale on $$ip... "; \
		ssh $(SSH_OPTS) -i $(SSH_KEY) $(SSH_USER)@$$ip " \
			if locale -a | grep -qi 'en_US.utf8'; then \
				echo 'already configured' >&2; \
			else \
				sudo locale-gen en_US.UTF-8 > /dev/null 2>&1; \
				sudo update-locale LANG=en_US.UTF-8 > /dev/null 2>&1; \
			fi" \
			&& echo "$(GREEN)✓$(NC)" || echo "$(RED)✗ Failed$(NC)"; \
	done

## Install k3s control plane
control-plane:
	@echo -e "$(BLUE)=== Installing k3s server on p1 ($(SERVER_IP)) ===$(NC)"
	@ssh $(SSH_OPTS) -i $(SSH_KEY) $(SSH_USER)@$(SERVER_IP) " \
		systemctl is-active --quiet k3s 2>/dev/null; \
		if [ \$$? -eq 0 ]; then \
			echo '$(YELLOW)k3s is already running$(NC)'; \
			exit 0; \
		fi; \
		curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION=$(K3S_VERSION) sh -s - server $(SERVER_INSTALL_OPTS)" \
		&& echo "$(GREEN)✓ Control plane installed$(NC)" || echo "$(RED)✗ Failed$(NC)"
	@echo "Waiting for k3s server to be ready..."
	@sleep $(K3S_STARTUP_WAIT)

## Install k3s worker nodes
workers:
	@echo -e "$(BLUE)=== Retrieving server token from p1 ===$(NC)"
	@K3S_TOKEN=$$(ssh $(SSH_OPTS) -i $(SSH_KEY) $(SSH_USER)@$(SERVER_IP) "sudo cat /var/lib/rancher/k3s/server/node-token"); \
	if [ -z "$$K3S_TOKEN" ]; then \
		echo -e "$(RED)✗ Failed to retrieve k3s token!$(NC)"; \
		exit 1; \
	fi; \
	echo -e "$(GREEN)✓ Token retrieved successfully$(NC)"; \
	echo -e "$(BLUE)=== Installing k3s agents on worker nodes ===$(NC)"; \
	for ip in $(AGENT_IPS); do \
		echo "Installing k3s agent on $$ip..."; \
		ssh $(SSH_OPTS) -i $(SSH_KEY) $(SSH_USER)@$$ip " \
			systemctl is-active --quiet k3s-agent 2>/dev/null; \
			if [ \$$? -eq 0 ]; then \
				echo '$(YELLOW)k3s-agent is already running$(NC)'; \
				exit 0; \
			fi; \
			curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION=$(K3S_VERSION) K3S_URL=https://$(SERVER_IP):6443 K3S_TOKEN=$$K3S_TOKEN sh -s - agent $(AGENT_INSTALL_OPTS)" \
			&& echo "$(GREEN)✓ Agent $$ip installed$(NC)" || echo "$(RED)✗ Failed$(NC)"; \
	done

## Install Cilium CNI
cilium:
	@echo -e "$(BLUE)=== Installing Cilium CNI ===$(NC)"
	@echo "Installing Cilium CLI..."
	@ssh $(SSH_OPTS) -i $(SSH_KEY) $(SSH_USER)@$(SERVER_IP) " \
		CILIUM_CLI_VERSION=\$$(curl -s https://raw.githubusercontent.com/cilium/cilium-cli/main/stable.txt); \
		CLI_ARCH=arm64; \
		curl -L --fail --remote-name-all https://github.com/cilium/cilium-cli/releases/download/\$${CILIUM_CLI_VERSION}/cilium-linux-\$${CLI_ARCH}.tar.gz{,.sha256sum} 2>/dev/null; \
		sha256sum --check cilium-linux-\$${CLI_ARCH}.tar.gz.sha256sum; \
		sudo tar xzvfC cilium-linux-\$${CLI_ARCH}.tar.gz /usr/local/bin > /dev/null; \
		rm cilium-linux-\$${CLI_ARCH}.tar.gz{,.sha256sum}" \
		&& echo "$(GREEN)✓ Cilium CLI installed$(NC)" || echo "$(RED)✗ Failed$(NC)"
	@echo "Installing Cilium on cluster..."
	@ssh $(SSH_OPTS) -i $(SSH_KEY) $(SSH_USER)@$(SERVER_IP) " \
		export KUBECONFIG=/etc/rancher/k3s/k3s.yaml; \
		cilium install --version 1.16.5" \
		&& echo "$(GREEN)✓ Cilium installed$(NC)" || echo "$(RED)✗ Failed$(NC)"
	@echo "Waiting for Cilium to be ready..."
	@ssh $(SSH_OPTS) -i $(SSH_KEY) $(SSH_USER)@$(SERVER_IP) " \
		export KUBECONFIG=/etc/rancher/k3s/k3s.yaml; \
		cilium status --wait" \
		&& echo "$(GREEN)✓ Cilium is ready$(NC)" || echo "$(RED)✗ Failed$(NC)"

## Download kubeconfig and add as 'pi' context
kubeconfig:
	@echo "$(BLUE)=== Downloading kubeconfig from control plane ===$(NC)"
	@command -v yq >/dev/null 2>&1 || { \
		echo "$(RED)✗ Error: yq is not installed$(NC)"; \
		echo "$(YELLOW)Install with:$(NC)"; \
		echo "  macOS: brew install yq"; \
		echo "  Linux: sudo wget https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64 -O /usr/local/bin/yq && sudo chmod +x /usr/local/bin/yq"; \
		exit 1; \
	}
	@scp $(SSH_OPTS) -i $(SSH_KEY) $(SSH_USER)@$(SERVER_IP):/etc/rancher/k3s/k3s.yaml k3s.yaml
	@yq '.clusters[0].cluster.certificate-authority-data' k3s.yaml | base64 -d > cluster-ca-data.crt
	@yq '.users[0].user.client-certificate-data' k3s.yaml | base64 -d > client-cert-data.crt
	@yq '.users[0].user.client-key-data' k3s.yaml | base64 -d > client-key-data.key
	@echo "$(BLUE)Configuring kubectl context 'pi'...$(NC)"
	@kubectl config set-cluster pi --server=https://$(SERVER_IP):6443
	@kubectl config set-cluster pi --embed-certs --certificate-authority='cluster-ca-data.crt'
	@kubectl config set-credentials pi --embed-certs --client-certificate='client-cert-data.crt'
	@kubectl config set-credentials pi --embed-certs --client-key='client-key-data.key'
	@kubectl config set-context pi --cluster='pi' --user='pi'
	@kubectl config use-context pi
	@rm -f k3s.yaml cluster-ca-data.crt client-cert-data.crt client-key-data.key
	@echo "$(GREEN)✓ Kubeconfig configured with context 'pi'$(NC)"
	@echo "$(BLUE)Test with: kubectl get nodes$(NC)"

## Uninstall k3s from worker nodes
uninstall-workers:
	@echo -e "$(BLUE)=== Uninstalling k3s from worker nodes ===$(NC)"
	@for ip in $(AGENT_IPS); do \
		echo "Uninstalling k3s-agent from $$ip..."; \
		ssh $(SSH_OPTS) -i $(SSH_KEY) $(SSH_USER)@$$ip " \
			if [ -f /usr/local/bin/k3s-agent-uninstall.sh ]; then \
				sudo /usr/local/bin/k3s-agent-uninstall.sh; \
				echo '$(GREEN)✓ k3s-agent uninstalled$(NC)'; \
			else \
				echo '$(YELLOW)k3s-agent not installed$(NC)'; \
			fi"; \
	done

## Uninstall k3s from control plane
uninstall-control-plane:
	@echo -e "$(BLUE)=== Uninstalling k3s from control plane ===$(NC)"
	@ssh $(SSH_OPTS) -i $(SSH_KEY) $(SSH_USER)@$(SERVER_IP) " \
		if [ -f /usr/local/bin/k3s-uninstall.sh ]; then \
			sudo /usr/local/bin/k3s-uninstall.sh; \
			echo '$(GREEN)✓ k3s-server uninstalled$(NC)'; \
		else \
			echo '$(YELLOW)k3s not installed$(NC)'; \
		fi"

## Uninstall k3s from all nodes
uninstall-all: uninstall-workers uninstall-control-plane
	@echo -e "$(GREEN)✓ k3s uninstalled from all nodes$(NC)"

## Check cluster status
status:
	@echo -e "$(BLUE)=== Cluster Status ===$(NC)"
	@ssh $(SSH_OPTS) -i $(SSH_KEY) $(SSH_USER)@$(SERVER_IP) "sudo kubectl get nodes -o wide" || echo -e "$(RED)✗ Cluster not accessible$(NC)"
	@echo -e ""
	@echo -e "$(BLUE)=== Cilium Status ===$(NC)"
	@ssh $(SSH_OPTS) -i $(SSH_KEY) $(SSH_USER)@$(SERVER_IP) "KUBECONFIG=/etc/rancher/k3s/k3s.yaml cilium status" || echo -e "$(YELLOW)Cilium not installed or not accessible$(NC)"

## Show help
help:
	@echo -e "$(BLUE)k3s Cluster Management Makefile$(NC)"
	@echo -e ""
	@echo -e "$(GREEN)Available targets:$(NC)"
	@echo -e "  $(YELLOW)make all$(NC)                    - Full cluster setup (default)"
	@echo -e "  $(YELLOW)make update-all$(NC)             - Update all nodes"
	@echo -e "  $(YELLOW)make configure-system$(NC)       - Configure system settings"
	@echo -e "  $(YELLOW)make control-plane$(NC)          - Install control plane only"
	@echo -e "  $(YELLOW)make workers$(NC)                - Install workers only"
	@echo -e "  $(YELLOW)make cilium$(NC)                 - Install Cilium CNI"
	@echo -e "  $(YELLOW)make uninstall-all$(NC)          - Uninstall k3s from all nodes"
	@echo -e "  $(YELLOW)make uninstall-control-plane$(NC) - Uninstall from control plane"
	@echo -e "  $(YELLOW)make uninstall-workers$(NC)      - Uninstall from workers"
	@echo -e "  $(YELLOW)make status$(NC)                 - Check cluster status"
	@echo -e "  $(YELLOW)make help$(NC)                   - Show this help message"
	@echo -e ""
	@echo -e "$(BLUE)Configuration:$(NC)"
	@echo -e "  Control Plane: p1 ($(SERVER_IP))"
	@echo -e "  Workers: p2 ($(word 1,$(AGENT_IPS))), p3 ($(word 2,$(AGENT_IPS)))"
	@echo -e "  k3s Version: $(K3S_VERSION)"
