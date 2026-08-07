(function () {
  "use strict";

  var HTML = document.documentElement;

  function applyStoredTheme() {
    var saved = localStorage.getItem("theme");
    if (saved === "dark" || saved === "light") {
      HTML.setAttribute("data-theme", saved);
    }
  }

  function toggleTheme() {
    var current = HTML.getAttribute("data-theme") || "auto";
    var next = current === "dark" ? "light" : "dark";
    HTML.setAttribute("data-theme", next);
    localStorage.setItem("theme", next);
  }

  function handleKeyboard(e) {
    if (e.target.matches("input, textarea")) return;
    if (e.key === "a") window.location.href = "/?filter=alerts";
    if (e.key === "e") window.location.href = "/";
  }

  var SCROLL_PIN_THRESHOLD = 48;

  function keepNewestVisible() {
    var feed = document.getElementById("feed");
    if (!feed) return;
    if (feed.scrollTop <= SCROLL_PIN_THRESHOLD) {
      feed.scrollTop = 0;
    }
  }

  function watchFeedForNewItems() {
    var feed = document.getElementById("feed");
    if (!feed) return;

    new MutationObserver(function (mutations) {
      for (var i = 0; i < mutations.length; i++) {
        if (mutations[i].addedNodes.length > 0) {
          keepNewestVisible();
          return;
        }
      }
    }).observe(feed, { childList: true });
  }

  function init() {
    applyStoredTheme();

    var toggle = document.querySelector(".theme-toggle");
    if (toggle) toggle.addEventListener("click", toggleTheme);

    document.addEventListener("keydown", handleKeyboard);

    watchFeedForNewItems();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
