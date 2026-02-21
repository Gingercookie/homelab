# Docker registry username - change this if your username changes
REGISTRY := wilgrimthepilgrim

# The target platform for your ARM-based homelab cluster
PLATFORM := linux/arm64

# List of all services - add new ones here as your project grows
SERVICES := api crew delivery package ship traffic

# The 'all' target builds and pushes every service in the SERVICES list.
# This is what runs when you just type 'make' with no arguments.
.PHONY: all $(SERVICES)
all: $(SERVICES)

# This is a pattern rule - it generates a target for each service name.
# When you run 'make delivery', Make expands this into:
#   docker buildx build --platform linux/arm64 -t wilgrimthepilgrim/delivery:latest ./delivery --push
$(SERVICES):
	docker buildx build --platform $(PLATFORM) -t $(REGISTRY)/$@:latest \
		-f ./$@/Dockerfile \
		--build-arg SERVICE=$@ \
		. --push

.PHONY: help
help:
	@echo "Usage:"
	@echo "  make              - build and push all services"
	@echo "  make <service>    - build and push a single service"
	@echo ""
	@echo "Services: $(SERVICES)"
	@echo ""
	@echo "Examples:"
	@echo "  make delivery"
	@echo "  make api crew"
