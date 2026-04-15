(function () {
  "use strict";

  function meta(name) {
    var el = document.querySelector('meta[name="' + name + '"]');
    return el ? el.content : "";
  }

  var kcURL = meta("keycloak-url");
  var kcRealm = meta("keycloak-realm");
  var kcClientId = meta("keycloak-client-id");

  var i18n = document.body.dataset || {};

  document.addEventListener("click", function (e) {
    var flag = e.target.closest("[data-lang]");
    if (!flag) return;
    e.preventDefault();
    document.cookie = "fresnel_lang=" + flag.dataset.lang + ";path=/;SameSite=Strict;Secure";
    location.reload();
  });

  if (!kcURL || !kcRealm || !kcClientId) {
    document.getElementById("app").textContent =
      i18n.i18nMissingKc || "Missing Keycloak configuration.";
    return;
  }

  if (typeof Keycloak === "undefined") {
    document.getElementById("app").textContent =
      i18n.i18nKcFailed || "Keycloak adapter failed to load.";
    return;
  }

  var kc = new Keycloak({
    url: kcURL,
    realm: kcRealm,
    clientId: kcClientId,
  });

  window._fresnel = { keycloak: kc };
  Object.defineProperty(window, '__fresnelToken', {
    get: function () { return kc.token; }
  });

  kc.init({
    onLoad: "login-required",
    pkceMethod: "S256",
    checkLoginIframe: false,
    responseMode: "fragment",
  })
    .then(function (authenticated) {
      if (!authenticated) return;

      // Strip OIDC callback parameters from the URL fragment
      if (window.location.hash && window.location.hash.indexOf("state=") !== -1) {
        history.replaceState(null, "", window.location.pathname + window.location.search);
      }

      onAuthenticated(kc);
    })
    .catch(function () {
      document.getElementById("app").textContent =
        i18n.i18nAuthFailed || "Authentication failed. Please refresh the page.";
    });

  function initSidebarToggle() {
    var toggle = document.getElementById("sidebar-toggle");
    var sidebar = document.getElementById("sidebar");
    var backdrop = document.getElementById("sidebar-backdrop");
    if (!toggle || !sidebar) return;

    function openSidebar() {
      sidebar.classList.add("open");
      if (backdrop) backdrop.classList.add("visible");
    }
    function closeSidebar() {
      sidebar.classList.remove("open");
      if (backdrop) backdrop.classList.remove("visible");
    }

    toggle.addEventListener("click", function () {
      if (sidebar.classList.contains("open")) {
        closeSidebar();
      } else {
        openSidebar();
      }
    });
    if (backdrop) {
      backdrop.addEventListener("click", closeSidebar);
    }
    sidebar.addEventListener("click", function (evt) {
      if (evt.target.closest(".nav-link, .nav-sublink")) {
        closeSidebar();
      }
    });
  }

  function initDataActions() {
    document.body.addEventListener("click", function (evt) {
      var el = evt.target.closest("[data-action]");
      if (!el) return;
      var action = el.getAttribute("data-action");

      switch (action) {
        case "toggle-tree":
          evt.stopPropagation();
          el.classList.toggle("open");
          var children = el.closest(".tree-node").querySelector(":scope > .tree-children");
          if (children) children.classList.toggle("collapsed");
          break;

        case "select-tree-item":
          document.querySelectorAll(".tree-selected").forEach(function (s) { s.classList.remove("tree-selected"); });
          el.classList.add("tree-selected");
          var panel = document.getElementById("side-panel");
          if (panel) {
            panel.classList.remove("side-panel-hidden");
            var layout = panel.closest(".dashboard-layout");
            if (layout) layout.classList.add("panel-open");
          }
          var titleEl = document.getElementById("side-panel-title");
          var suffix = i18n.i18nTimelineSuffix || "\u2014 Timeline";
          if (titleEl) titleEl.textContent = (el.getAttribute("data-name") || "") + " " + suffix;
          break;

        case "close-side-panel":
          var sp = document.getElementById("side-panel");
          if (sp) {
            sp.classList.add("side-panel-hidden");
            var spLayout = sp.closest(".dashboard-layout");
            if (spLayout) spLayout.classList.remove("panel-open");
          }
          document.querySelectorAll(".tree-selected").forEach(function (s) { s.classList.remove("tree-selected"); });
          break;

        case "reload-page":
          location.reload();
          break;

        case "show-link-event-form":
          var lf = document.getElementById("link-event-form");
          if (lf) lf.style.display = "block";
          break;

        case "hide-closest-card":
          var card = el.closest(".card");
          if (card) card.style.display = "none";
          break;

        case "show-correlation-form":
          var cf = document.getElementById("add-correlation-form");
          if (cf) cf.style.display = "block";
          el.style.display = "none";
          break;

        case "stop-propagation":
          evt.stopPropagation();
          break;

        case "start-campaign-selection":
          if (window.__fresnelStartCampaignSelection) window.__fresnelStartCampaignSelection();
          break;
        case "show-campaign-form":
          if (window.__fresnelShowCampaignForm) window.__fresnelShowCampaignForm();
          break;
        case "cancel-campaign-selection":
          if (window.__fresnelCancelCampaignSelection) window.__fresnelCancelCampaignSelection();
          break;
        case "submit-campaign":
          if (window.__fresnelSubmitCampaign) window.__fresnelSubmitCampaign();
          break;
      }
    });

    document.body.addEventListener("click", function (evt) {
      var eventLink = evt.target.closest("[data-event-link]");
      if (eventLink) {
        evt.preventDefault();
        var eid = eventLink.getAttribute("data-event-link");
        htmx.ajax("GET", "/api/v1/events/" + eid, { target: "#app", swap: "innerHTML" });
        history.pushState({}, "", "/events/" + eid);
      }
    });

    document.body.addEventListener("change", function (evt) {
      var el = evt.target.closest("[data-action]");
      if (!el) return;
      var action = el.getAttribute("data-action");

      switch (action) {
        case "update-selection-count":
          if (window.__fresnelUpdateSelectionCount) window.__fresnelUpdateSelectionCount();
          break;
        case "toggle-tlp-red":
          var recipients = document.getElementById("tlp-red-recipients");
          if (recipients) recipients.style.display = el.value === "RED" ? "block" : "none";
          break;
      }
    });
  }

  function onAuthenticated(kc) {
    updateUserInfo(kc);
    initSidebarToggle();
    initDataActions();

    document.body.addEventListener("htmx:configRequest", function (evt) {
      evt.detail.headers["Authorization"] = "Bearer " + kc.token;
    });

    document.body.addEventListener("htmx:responseError", function (evt) {
      if (evt.detail.xhr && evt.detail.xhr.status === 401) {
        kc.updateToken(5).catch(function () {
          kc.login();
        });
      }
    });

    setInterval(function () {
      kc.updateToken(60).catch(function () {
        kc.login();
      });
    }, 30000);

    loadNav();
    loadOrgContext(kc);
    loadInitialContent();

    window.addEventListener("popstate", function () {
      loadContentForPath(window.location.pathname);
    });

    document.body.addEventListener("htmx:pushedIntoHistory", function (evt) {
      highlightActiveNav(evt.detail.path);
    });
  }

  function pathToAPI(path, search) {
    var base;
    if (path === "/" || path === "") {
      base = "/api/v1/dashboard";
    } else {
      base = "/api/v1" + path.replace(/\/$/, "");
    }
    return search ? base + search : base;
  }

  function loadInitialContent() {
    var path = window.location.pathname;
    var search = window.location.search;
    htmx.ajax("GET", pathToAPI(path, search), { target: "#app", swap: "innerHTML" });
    highlightActiveNav(path);
  }

  function loadContentForPath(path) {
    var search = window.location.search;
    htmx.ajax("GET", pathToAPI(path, search), { target: "#app", swap: "innerHTML" });
    highlightActiveNav(path);
  }

  function loadNav() {
    htmx.ajax("GET", "/api/v1/nav", { target: "#sidebar", swap: "innerHTML" });
  }

  function loadOrgContext(kc) {
    var selector = document.getElementById("org-context-selector");
    if (!selector) return;

    fetch("/api/v1/users/me", {
      headers: {
        Authorization: "Bearer " + kc.token,
        Accept: "application/json",
      },
    })
      .then(function (r) { return r.json(); })
      .then(function (data) {
        if (!data.org_memberships || data.org_memberships.length < 2) return;

        selector.style.display = "";
        selector.innerHTML = "";
        data.org_memberships.forEach(function (org) {
          var opt = document.createElement("option");
          opt.value = org.id;
          opt.textContent = org.name;
          if (org.id === data.active_org_context) opt.selected = true;
          selector.appendChild(opt);
        });

        selector.addEventListener("change", function () {
          fetch("/api/v1/users/me/org-context", {
            method: "PUT",
            headers: {
              Authorization: "Bearer " + kc.token,
              "Content-Type": "application/json",
            },
            body: JSON.stringify({ org_id: selector.value }),
          }).then(function () {
            loadContentForPath(window.location.pathname);
            loadNav();
          });
        });
      })
      .catch(function () {});
  }

  function highlightActiveNav(path) {
    var sidebar = document.getElementById("sidebar");
    if (!sidebar) return;
    var links = sidebar.querySelectorAll(".nav-link");
    links.forEach(function (link) {
      var pushUrl = link.getAttribute("hx-push-url");
      if (pushUrl && path.indexOf(pushUrl) === 0) {
        link.classList.add("active");
      } else {
        link.classList.remove("active");
      }
    });
  }

  function updateUserInfo(kc) {
    var info = document.getElementById("user-info");
    if (!info) return;

    var tp = kc.tokenParsed || {};
    var name = tp.name || tp.preferred_username || "";
    var nameSpan = document.createElement("span");
    nameSpan.textContent = name;

    var logout = document.createElement("a");
    logout.href = "#";
    logout.textContent = i18n.i18nLogout || "Log out";
    logout.addEventListener("click", function (e) {
      e.preventDefault();
      kc.logout({ redirectUri: window.location.origin + "/" });
    });

    info.textContent = "";
    info.appendChild(nameSpan);
    info.appendChild(document.createTextNode(" "));
    info.appendChild(logout);
  }
})();
