(function () {
  "use strict";

  function meta(name) {
    var el = document.querySelector('meta[name="' + name + '"]');
    return el ? el.content : "";
  }

  var kcURL = meta("keycloak-url");
  var kcRealm = meta("keycloak-realm");
  var kcClientId = meta("keycloak-client-id");

  if (!kcURL || !kcRealm || !kcClientId) {
    document.getElementById("app").textContent =
      "Missing Keycloak configuration.";
    return;
  }

  if (typeof Keycloak === "undefined") {
    document.getElementById("app").textContent =
      "Keycloak adapter failed to load.";
    return;
  }

  var kc = new Keycloak({
    url: kcURL,
    realm: kcRealm,
    clientId: kcClientId,
  });

  window._fresnel = { keycloak: kc };

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
        "Authentication failed. Please refresh the page.";
    });

  function onAuthenticated(kc) {
    updateUserInfo(kc);

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

  function pathToAPI(path) {
    if (path === "/" || path === "") return "/api/v1/dashboard";
    return "/api/v1" + path.replace(/\/$/, "");
  }

  function loadInitialContent() {
    var path = window.location.pathname;
    htmx.ajax("GET", pathToAPI(path), { target: "#app", swap: "innerHTML" });
    highlightActiveNav(path);
  }

  function loadContentForPath(path) {
    htmx.ajax("GET", pathToAPI(path), { target: "#app", swap: "innerHTML" });
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
    logout.textContent = "Log out";
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
