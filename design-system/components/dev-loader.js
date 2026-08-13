/* Dev fallback loader — used ONLY while the auto-generated _ds_bundle.js is absent.
   1) Builds window.VergeDS from the component sources (skipped if a real bundle
      namespace is already present).
   2) Exposes __runJsx(paths) / __runJsxTags() so pages can execute JSX through
      Babel.transform directly (babel-standalone's own text/babel runner is not
      used anywhere in this project).
   Include AFTER react + babel-standalone:
   <script src=".../components/dev-loader.js" data-base=".../"></script> */
(function () {
  if (!window.Babel || !window.React) { console.error("dev-loader: load react + babel-standalone first"); return; }
  var cur = document.currentScript;
  var base = (cur && cur.getAttribute("data-base")) || "";

  function fetchSync(url) {
    var x = new XMLHttpRequest();
    x.open("GET", url, false);
    x.send();
    if (x.status >= 200 && x.status < 300) return x.responseText;
    throw new Error("dev-loader: failed to fetch " + url);
  }
  // Load the compiler-generated bundle if it exists (it defines the real namespace).
  try {
    var bx = new XMLHttpRequest();
    bx.open("GET", base + "_ds_bundle.js", false);
    bx.send();
    if (bx.status >= 200 && bx.status < 300 && bx.responseText) new Function(bx.responseText)();
  } catch (e) {}
  function compile(src, filename) {
    return Babel.transform(src, { presets: [["env", { modules: "commonjs" }], ["react", { runtime: "classic" }]], filename: filename }).code;
  }

  var mods = {};
  function normalize(from, req) {
    var dir = from.split("/").slice(0, -1).join("/");
    var parts = (dir + "/" + req.replace(/\.jsx$/, "")).split("/");
    var st = [];
    parts.forEach(function (s) { if (s === "." || s === "") return; if (s === "..") st.pop(); else st.push(s); });
    return st.join("/");
  }
  var srcs = {};
  function loadComponent(key) {
    if (mods[key]) return mods[key].exports;
    var m = { exports: {} };
    mods[key] = m;
    new Function("require", "module", "exports", compile(srcs[key], key + ".jsx"))(
      function (r) { return r === "react" ? window.React : loadComponent(normalize(key, r)); }, m, m.exports
    );
    return m.exports;
  }

  /* Run project-relative .jsx files (no imports; they use window globals). */
  window.__runJsx = function (paths) {
    paths.forEach(function (p) {
      var m = { exports: {} };
      new Function("require", "module", "exports", compile(fetchSync(p), p))(
        function (r) { return r === "react" ? window.React : undefined; }, m, m.exports
      );
    });
  };
  /* Run inline <script type="text/jsx-run"> blocks in document order. */
  window.__runJsxTags = function () {
    var tags = document.querySelectorAll('script[type="text/jsx-run"]');
    Array.prototype.forEach.call(tags, function (s, i) {
      var m = { exports: {} };
      new Function("require", "module", "exports", compile(s.textContent, "inline-" + i + ".jsx"))(
        function (r) { return r === "react" ? window.React : undefined; }, m, m.exports
      );
    });
  };

  // Skip namespace building if the real bundle already provided one.
  try {
    for (var k in window) {
      var w = window[k];
      if (w && typeof w === "object" && w.Button && w.Table && w.TopNav) return;
    }
  } catch (e) {}

  var FILES = [
    "forms/Button", "forms/IconButton", "forms/Input", "forms/Select", "forms/Checkbox", "forms/Radio", "forms/Switch",
    "display/Card", "display/Badge", "display/SeverityBadge", "display/Tag", "display/Stat", "display/StatusDot", "display/Table", "display/Wordmark",
    "feedback/Dialog", "feedback/Toast", "feedback/Tooltip", "feedback/EmptyState",
    "navigation/TopNav", "navigation/Tabs", "navigation/Footer",
  ];
  FILES.forEach(function (p) { srcs[p] = fetchSync(base + "components/" + p + ".jsx"); });
  var NS = {};
  FILES.forEach(function (p) {
    var e = loadComponent(p);
    for (var n in e) if (n !== "__esModule" && n !== "default") NS[n] = e[n];
  });
  window.VergeDS = NS;
  console.log("dev-loader: VergeDS ready with " + Object.keys(NS).length + " components");
})();
