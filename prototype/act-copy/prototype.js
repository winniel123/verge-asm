// Prototype chrome. Not shipped, not ported.

function show(site, v) {
  site.querySelectorAll('.pt-variant').forEach(function (el) { el.hidden = el.dataset.v !== v; });
  site.querySelectorAll('.pt-switch button').forEach(function (b) {
    b.setAttribute('aria-pressed', String(b.dataset.v === v));
  });
  site.querySelectorAll('[data-note]').forEach(function (el) { el.hidden = el.dataset.note !== v; });
}

document.querySelectorAll('.pt-site').forEach(function (site) {
  site.querySelectorAll('.pt-switch button').forEach(function (b) {
    b.addEventListener('click', function () { show(site, b.dataset.v); });
  });
  var pressed = site.querySelector('.pt-switch button[aria-pressed="true"]');
  show(site, pressed ? pressed.dataset.v : 'a');
});

// Reading all five wrong strings in one pass is the comparison the ticket is really about.
document.getElementById('pt-all-today').addEventListener('click', function () {
  document.querySelectorAll('.pt-site').forEach(function (site) { show(site, 'today'); });
});
document.getElementById('pt-all-chosen').addEventListener('click', function () {
  document.querySelectorAll('.pt-site').forEach(function (site) { show(site, 'a'); });
});

var theme = document.getElementById('pt-theme');
theme.addEventListener('click', function () {
  var dark = document.documentElement.getAttribute('data-theme') === 'dark';
  document.documentElement.setAttribute('data-theme', dark ? 'light' : 'dark');
  theme.textContent = dark ? 'Dark' : 'Light';
});
