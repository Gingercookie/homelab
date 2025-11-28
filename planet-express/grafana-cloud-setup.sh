#!/bin/bash

helm repo add grafana https://grafana.github.io/helm-charts &&
  helm repo update &&
  helm upgrade --install --atomic --timeout 300s grafana-k8s-monitoring grafana/k8s-monitoring \
    --namespace "monitoring" --create-namespace --values - <<'EOF'
cluster:
  name: pi
destinations:
  - name: grafana-cloud-metrics
    type: prometheus
    url: https://prometheus-prod-36-prod-us-west-0.grafana.net./api/prom/push
    auth:
      type: basic
      username: "2596801"
      password: <REDACTED>
  - name: grafana-cloud-logs
    type: loki
    url: https://logs-prod-021.grafana.net./loki/api/v1/push
    auth:
      type: basic
      username: "1294077"
      password: <REDACTED>
  - name: grafana-cloud-otlp-endpoint
    type: otlp
    url: https://otlp-gateway-prod-us-west-0.grafana.net./otlp
    protocol: http
    auth:
      type: basic
      username: "1336283"
      password: <REDACTED>
    metrics:
      enabled: true
    logs:
      enabled: true
    traces:
      enabled: true
clusterMetrics:
  enabled: true
  opencost:
    enabled: true
    metricsSource: grafana-cloud-metrics
    opencost:
      exporter:
        defaultClusterId: pi
      prometheus:
        existingSecretName: grafana-cloud-metrics-grafana-k8s-monitoring
        external:
          url: https://prometheus-prod-36-prod-us-west-0.grafana.net./api/prom
  kepler:
    enabled: true
clusterEvents:
  enabled: true
podLogs:
  enabled: true
applicationObservability:
  enabled: true
  receivers:
    otlp:
      grpc:
        enabled: true
        port: 4317
      http:
        enabled: true
        port: 4318
    zipkin:
      enabled: true
      port: 9411
alloy-metrics:
  enabled: true
  alloy:
    extraEnv:
      - name: GCLOUD_RW_API_KEY
        valueFrom:
          secretKeyRef:
            name: alloy-metrics-remote-cfg-grafana-k8s-monitoring
            key: password
      - name: CLUSTER_NAME
        value: pi
      - name: NAMESPACE
        valueFrom:
          fieldRef:
            fieldPath: metadata.namespace
      - name: POD_NAME
        valueFrom:
          fieldRef:
            fieldPath: metadata.name
      - name: GCLOUD_FM_COLLECTOR_ID
        value: grafana-k8s-monitoring-$(CLUSTER_NAME)-$(NAMESPACE)-$(POD_NAME)
  remoteConfig:
    enabled: true
    url: https://fleet-management-prod-014.grafana.net
    auth:
      type: basic
      username: "1336283"
      password: <REDACTED>
alloy-singleton:
  enabled: true
  alloy:
    extraEnv:
      - name: GCLOUD_RW_API_KEY
        valueFrom:
          secretKeyRef:
            name: alloy-singleton-remote-cfg-grafana-k8s-monitoring
            key: password
      - name: CLUSTER_NAME
        value: pi
      - name: NAMESPACE
        valueFrom:
          fieldRef:
            fieldPath: metadata.namespace
      - name: POD_NAME
        valueFrom:
          fieldRef:
            fieldPath: metadata.name
      - name: GCLOUD_FM_COLLECTOR_ID
        value: grafana-k8s-monitoring-$(CLUSTER_NAME)-$(NAMESPACE)-$(POD_NAME)
  remoteConfig:
    enabled: true
    url: https://fleet-management-prod-014.grafana.net
    auth:
      type: basic
      username: "1336283"
      password: <REDACTED>
alloy-logs:
  enabled: true
  alloy:
    extraEnv:
      - name: GCLOUD_RW_API_KEY
        valueFrom:
          secretKeyRef:
            name: alloy-logs-remote-cfg-grafana-k8s-monitoring
            key: password
      - name: CLUSTER_NAME
        value: pi
      - name: NAMESPACE
        valueFrom:
          fieldRef:
            fieldPath: metadata.namespace
      - name: POD_NAME
        valueFrom:
          fieldRef:
            fieldPath: metadata.name
      - name: NODE_NAME
        valueFrom:
          fieldRef:
            fieldPath: spec.nodeName
      - name: GCLOUD_FM_COLLECTOR_ID
        value: grafana-k8s-monitoring-$(CLUSTER_NAME)-$(NAMESPACE)-alloy-logs-$(NODE_NAME)
  remoteConfig:
    enabled: true
    url: https://fleet-management-prod-014.grafana.net
    auth:
      type: basic
      username: "1336283"
      password: <REDACTED>
alloy-receiver:
  enabled: true
  alloy:
    extraPorts:
      - name: otlp-grpc
        port: 4317
        targetPort: 4317
        protocol: TCP
      - name: otlp-http
        port: 4318
        targetPort: 4318
        protocol: TCP
      - name: zipkin
        port: 9411
        targetPort: 9411
        protocol: TCP
    extraEnv:
      - name: GCLOUD_RW_API_KEY
        valueFrom:
          secretKeyRef:
            name: alloy-receiver-remote-cfg-grafana-k8s-monitoring
            key: password
      - name: CLUSTER_NAME
        value: pi
      - name: NAMESPACE
        valueFrom:
          fieldRef:
            fieldPath: metadata.namespace
      - name: POD_NAME
        valueFrom:
          fieldRef:
            fieldPath: metadata.name
      - name: NODE_NAME
        valueFrom:
          fieldRef:
            fieldPath: spec.nodeName
      - name: GCLOUD_FM_COLLECTOR_ID
        value: grafana-k8s-monitoring-$(CLUSTER_NAME)-$(NAMESPACE)-alloy-receiver-$(NODE_NAME)
  remoteConfig:
    enabled: true
    url: https://fleet-management-prod-014.grafana.net
    auth:
      type: basic
      username: "1336283"
      password: <REDACTED>
EOF
