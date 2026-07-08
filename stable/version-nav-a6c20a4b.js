// SPDX-License-Identifier: GPL-3.0-or-later

(function () {
  var publishedBase = "/idp-register/";
  var href = window.location.pathname.indexOf(publishedBase) === 0 ? publishedBase : "/";

  var link = document.createElement("a");
  link.href = href;
  link.textContent = "Versions";
  link.className = "icon-button";
  link.title = "Documentation versions";
  link.setAttribute("aria-label", "Documentation versions");

  var menuBar = document.querySelector(".menu-bar");
  if (menuBar) {
    menuBar.appendChild(link);
    return;
  }

  var sidebar = document.querySelector(".sidebar");
  if (sidebar) {
    var item = document.createElement("div");
    item.appendChild(link);
    sidebar.insertBefore(item, sidebar.firstChild);
  }
})();
