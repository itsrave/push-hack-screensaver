package main

// indexHTML is the browser config page served at /. It's a self-contained page
// (no assets, no deps) that drives the existing /api/config GET/POST endpoints.
var indexHTML = []byte(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Push Screensaver</title>
<style>
  :root { color-scheme: dark; }
  body { margin:0; font:15px/1.5 system-ui,sans-serif; background:#111; color:#eee;
         display:flex; justify-content:center; }
  main { width:100%; max-width:420px; padding:24px 20px 40px; }
  h1 { font-size:20px; margin:0 0 4px; }
  .status { color:#999; font-size:13px; margin-bottom:24px; }
  .status .dot { display:inline-block; width:8px; height:8px; border-radius:50%;
                 background:#555; margin-right:6px; vertical-align:middle; }
  .status.active .dot { background:#4ade80; }
  label { display:block; margin:18px 0 6px; font-weight:600; }
  .val { float:right; color:#999; font-weight:400; }
  select, input[type=range] { width:100%; }
  select { padding:8px; background:#1c1c1c; color:#eee; border:1px solid #333; border-radius:6px; }
  .row { display:flex; align-items:center; justify-content:space-between; margin:18px 0 6px; }
  .row label { margin:0; }
  .switch { position:relative; width:46px; height:26px; }
  .switch input { opacity:0; width:0; height:0; }
  .slider { position:absolute; inset:0; background:#444; border-radius:26px; transition:.2s; cursor:pointer; }
  .slider:before { content:""; position:absolute; height:20px; width:20px; left:3px; top:3px;
                   background:#eee; border-radius:50%; transition:.2s; }
  input:checked + .slider { background:#4ade80; }
  input:checked + .slider:before { transform:translateX(20px); }
</style>
</head>
<body>
<main>
  <h1>Screensaver</h1>
  <div class="status" id="status"><span class="dot"></span><span id="statustext">…</span></div>

  <div class="row">
    <label for="enabled">Enabled</label>
    <span class="switch"><input type="checkbox" id="enabled"><span class="slider"></span></span>
  </div>

  <label for="animation">Animation</label>
  <select id="animation"></select>

  <label for="idle">Idle timeout (seconds)</label>
  <input type="number" id="idle" min="5" max="3600" step="1"
         style="width:100%;padding:8px;background:#1c1c1c;color:#eee;border:1px solid #333;border-radius:6px;box-sizing:border-box">

  <label for="speed">Speed <span class="val" id="speedval"></span></label>
  <input type="range" id="speed" min="1" max="10" step="1">
</main>
<script>
const $ = id => document.getElementById(id);
let animsLoaded = false;

function render(s) {
  $('enabled').checked = s.enabled;
  if (!animsLoaded) {
    $('animation').innerHTML = s.animations
      .map((n,i) => '<option value="'+i+'">'+n+'</option>').join('');
    animsLoaded = true;
  }
  $('animation').value = s.animation;
  if (document.activeElement !== $('idle')) $('idle').value = s.idle_seconds;
  $('speed').value = s.speed; $('speedval').textContent = s.speed;
  const st = $('status');
  st.classList.toggle('active', s.active);
  $('statustext').textContent = s.active ? 'active now' : 'idle — waiting';
}

async function load() {
  render(await (await fetch('/api/config')).json());
}
async function patch(body) {
  render(await (await fetch('/api/config',
    {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(body)}
  )).json());
}

$('enabled').onchange  = e => patch({enabled: e.target.checked});
$('animation').onchange = e => patch({animation: +e.target.value});
$('idle').onchange = e => patch({idle_seconds: +e.target.value});
$('speed').oninput  = e => $('speedval').textContent = e.target.value;
$('speed').onchange = e => patch({speed: +e.target.value});

load();
setInterval(load, 3000); // reflect live active-state changes
</script>
</body>
</html>
`)
