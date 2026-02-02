"use strict";

const HomePager = (() => {
  const ANNOTATION_PREFIX = "homepage.link/";
  const API_ENDPOINT = "/api/ingresses";
  const REFRESH_INTERVAL = 30000;

  let refreshTimer = null;

  const elements = {
    appsContainer: null,
  };

  function init() {
    cacheElements();
    loadApplications();
    startAutoRefresh();
  }

  function cacheElements() {
    elements.appsContainer = document.getElementById("apps-container");
  }

  function startAutoRefresh() {
    if (refreshTimer) {
      clearInterval(refreshTimer);
    }
    refreshTimer = setInterval(loadApplications, REFRESH_INTERVAL);
  }

  async function loadApplications() {
    try {
      const response = await fetch(API_ENDPOINT);

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const data = await response.json();
      const apps = extractApplicationsFromIngresses(data.items || []);

      if (apps.length === 0) {
        renderEmptyState("No applications found with homepage annotations");
        return;
      }

      renderApplications(apps);
    } catch (error) {
      console.error("Failed to load applications:", error);
      renderEmptyState(`Failed to load applications: ${error.message}`);
    }
  }

  function extractApplicationsFromIngresses(ingresses) {
    const apps = [];

    for (const ingress of ingresses) {
      const app = buildApplicationFromIngress(ingress);
      if (app) {
        apps.push(app);
      }
    }

    return apps;
  }

  function buildApplicationFromIngress(ingress) {
    const metadata = ingress.metadata || {};
    const annotations = metadata.annotations || {};
    const spec = ingress.spec || {};

    if (annotations[`${ANNOTATION_PREFIX}enabled`] !== "true") {
      return null;
    }

    const host = getHost(annotations, spec);
    if (!host) {
      return null;
    }

    const hasTls = Array.isArray(spec.tls) && spec.tls.length > 0;
    const protocol = hasTls ? "https" : "http";

    return {
      name: annotations[`${ANNOTATION_PREFIX}name`] || metadata.name || "Unknown",
      namespace: metadata.namespace || "default",
      url: `${protocol}://${host}`,
      icon: annotations[`${ANNOTATION_PREFIX}icon`] || "🌐",
      description: annotations[`${ANNOTATION_PREFIX}description`] || "",
      resourceName: metadata.name || "unknown",
    };
  }

  function getHost(annotations, spec) {
    const customHost = annotations[`${ANNOTATION_PREFIX}host`];
    if (customHost) {
      return customHost;
    }

    const rules = spec.rules || [];
    if (rules.length > 0 && rules[0].host) {
      return rules[0].host;
    }

    return null;
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
