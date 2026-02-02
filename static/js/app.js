"use strict";

const HomePager = (() => {
  const ANNOTATION_PREFIX = "homepage.link/";

  const SAMPLE_YAML = `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: grafana
  namespace: monitoring
  annotations:
    homepage.link/enabled: "true"
    homepage.link/name: "Grafana"
    homepage.link/description: "Metrics & Dashboards"
    homepage.link/icon: "📊"
    homepage.link/internal-host: "grafana.internal.local"
    homepage.link/external-host: "grafana.example.com"
spec:
  rules:
  - host: grafana.internal.local
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: grafana
            port:
              number: 3000
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: prometheus
  namespace: monitoring
  annotations:
    homepage.link/enabled: "true"
    homepage.link/name: "Prometheus"
    homepage.link/icon: "🔥"
    homepage.link/internal-host: "prometheus.internal.local"
    homepage.link/external-host: "prometheus.example.com"
spec:
  rules:
  - host: prometheus.internal.local
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: jenkins
  namespace: ci-cd
  annotations:
    homepage.link/enabled: "true"
    homepage.link/name: "Jenkins CI/CD"
    homepage.link/icon: "⚙️"
    homepage.link/internal-host: "jenkins.internal.local"
    homepage.link/external-host: "jenkins.example.com"
spec:
  rules:
  - host: jenkins.internal.local
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: argocd
  namespace: argocd
  annotations:
    homepage.link/enabled: "true"
    homepage.link/name: "ArgoCD"
    homepage.link/description: "GitOps Continuous Delivery"
    homepage.link/icon: "🐙"
    homepage.link/internal-host: "argocd.internal.local"
    homepage.link/external-host: "argocd.example.com"
spec:
  tls:
  - hosts:
    - argocd.internal.local
  rules:
  - host: argocd.internal.local`;

  const elements = {
    yamlInput: null,
    appsContainer: null,
    loadButton: null,
    networkRadios: null,
  };

  function init() {
    cacheElements();
    bindEvents();
    loadSampleData();
  }

  function cacheElements() {
    elements.yamlInput = document.getElementById("yaml-input");
    elements.appsContainer = document.getElementById("apps-container");
    elements.loadButton = document.getElementById("load-button");
    elements.networkRadios = document.querySelectorAll('input[name="network"]');
  }

  function bindEvents() {
    elements.loadButton?.addEventListener("click", handleLoadClick);

    elements.networkRadios.forEach((radio) => {
      radio.addEventListener("change", handleNetworkChange);
    });

    elements.yamlInput?.addEventListener("keydown", handleKeyDown);
  }

  function handleLoadClick() {
    parseAndRender();
  }

  function handleNetworkChange() {
    const yamlValue = elements.yamlInput?.value.trim();
    if (yamlValue) {
      parseAndRender();
    }
  }

  function handleKeyDown(event) {
    if ((event.ctrlKey || event.metaKey) && event.key === "Enter") {
      event.preventDefault();
      parseAndRender();
    }
  }

  function loadSampleData() {
    if (elements.yamlInput) {
      elements.yamlInput.value = SAMPLE_YAML;
      parseAndRender();
    }
  }

  function getSelectedNetworkMode() {
    const checked = document.querySelector('input[name="network"]:checked');
    return checked?.value || "internal";
  }

  function parseAndRender() {
    const yaml = elements.yamlInput?.value.trim();
    const networkMode = getSelectedNetworkMode();

    if (!yaml) {
      renderEmptyState("Please paste your Kubernetes resources above");
      return;
    }

    try {
      const apps = extractApplications(yaml, networkMode);

      if (apps.length === 0) {
        renderEmptyState("No applications found with homepage annotations");
        return;
      }

      renderApplications(apps);
    } catch (error) {
      console.error("YAML parsing error:", error);
      renderEmptyState(`Error parsing YAML: ${error.message}`);
    }
  }

  function extractApplications(yaml, networkMode) {
    const resources = yaml.split("---");
    const apps = [];

    for (const resource of resources) {
      if (!resource.trim()) continue;

      const parsed = parseResource(resource);
      const app = buildApplication(parsed, networkMode);

      if (app) {
        apps.push(app);
      }
    }

    return apps;
  }

  function parseResource(resource) {
    const lines = resource.split("\n");
    const result = {
      metadata: {},
      annotations: {},
      spec: { host: null, hasTls: false },
    };

    let section = null;
    let inAnnotations = false;

    for (const line of lines) {
      const trimmed = line.trim();

      if (trimmed === "metadata:") {
        section = "metadata";
        inAnnotations = false;
        continue;
      }

      if (trimmed === "spec:") {
        section = "spec";
        inAnnotations = false;
        continue;
      }

      if (section === "metadata" && trimmed === "annotations:") {
        inAnnotations = true;
        continue;
      }

      if (section === "metadata" && !inAnnotations) {
        const match = trimmed.match(/^(\w+):\s*(.+)$/);
        if (match) {
          result.metadata[match[1]] = stripQuotes(match[2]);
        }
      }

      if (inAnnotations && trimmed.startsWith(ANNOTATION_PREFIX)) {
        const match = trimmed.match(/^homepage\.link\/([^:]+):\s*(.+)$/);
        if (match) {
          result.annotations[match[1]] = stripQuotes(match[2]);
        }
      }

      if (section === "spec") {
        if (trimmed.includes("host:") && !trimmed.startsWith("- hosts:")) {
          const match = trimmed.match(/host:\s*(.+)$/);
          if (match) {
            result.spec.host = stripQuotes(match[1]);
          }
        }

        if (trimmed === "tls:" || trimmed.startsWith("- hosts:")) {
          result.spec.hasTls = true;
        }
      }
    }

    return result;
  }

  function stripQuotes(value) {
    return value.replace(/^['"]|['"]$/g, "");
  }

  function buildApplication(parsed, networkMode) {
    const { metadata, annotations, spec } = parsed;

    if (annotations.enabled !== "true") {
      return null;
    }

    let host = null;
    let ingressType = networkMode;

    if (networkMode === "internal") {
      host = annotations["internal-host"] || spec.host;
    } else {
      host = annotations["external-host"] || spec.host;
    }

    if (!host) {
      return null;
    }

    const protocol = spec.hasTls || host.includes("443") ? "https" : "http";

    return {
      name: annotations.name || metadata.name || "Unknown",
      namespace: metadata.namespace || "default",
      url: `${protocol}://${host}`,
      icon: annotations.icon || "🌐",
      description: annotations.description || "",
      ingressType,
      resourceName: metadata.name || "unknown",
    };
  }

  function renderApplications(apps) {
    if (!elements.appsContainer) return;

    const fragment = document.createDocumentFragment();
    const grid = document.createElement("div");
    grid.className = "apps-grid";
    grid.setAttribute("role", "list");

    for (const app of apps) {
      grid.appendChild(createAppCard(app));
    }

    fragment.appendChild(grid);
    elements.appsContainer.innerHTML = "";
    elements.appsContainer.appendChild(fragment);
  }

  function createAppCard(app) {
    const card = document.createElement("a");
    card.href = app.url;
    card.className = "app-card";
    card.target = "_blank";
    card.rel = "noopener noreferrer";
    card.setAttribute("role", "listitem");

    card.innerHTML = `
      <div class="app-card__icon" aria-hidden="true">${escapeHtml(app.icon)}</div>
      <h2 class="app-card__name">${escapeHtml(app.name)}</h2>
      <p class="app-card__namespace">namespace: ${escapeHtml(app.namespace)}</p>
      ${app.description ? `<p class="app-card__description">${escapeHtml(app.description)}</p>` : ""}
      <p class="app-card__url">${escapeHtml(app.url)}</p>
      <div class="app-card__meta">
        <span class="badge badge--${app.ingressType}">${escapeHtml(app.ingressType)}</span>
        <p class="app-card__resource">Resource: ${escapeHtml(app.resourceName)}</p>
      </div>
    `;

    return card;
  }

  function renderEmptyState(message) {
    if (!elements.appsContainer) return;

    elements.appsContainer.innerHTML = `
      <div class="empty-state" role="status" aria-live="polite">
        <div class="empty-state__icon" aria-hidden="true">📦</div>
        <p class="empty-state__message">${escapeHtml(message)}</p>
      </div>
    `;
  }

  function escapeHtml(text) {
    const div = document.createElement("div");
    div.textContent = text;
    return div.innerHTML;
  }

  return { init };
})();

document.addEventListener("DOMContentLoaded", HomePager.init);
