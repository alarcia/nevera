// Admin screen: a full-screen dialog filled from /admin, so nothing navigates
// or reloads. Every card is a form saved in the background; the server answers
// with the card (or the items block) drawn again, or with a message to show.
// Closing after a change reloads the page so the fridge reflects it.
(() => {
  const dlg = document.getElementById("admin");
  const opener = document.getElementById("admin-open");
  if (!dlg || !opener) return;
  const host = document.getElementById("adm-in");
  const d = dlg.dataset;
  let dirty = false;
  let picking = null; // the card whose look the picker is editing

  const $ = (s) => dlg.querySelector(s);
  const fold = (s) => s.normalize("NFD").replace(/\p{M}/gu, "").toLowerCase();
  const sprite = (shape, skin) => `it k-${shape || "jar"}` + (skin ? ` c-${skin}` : "");
  const say = (msg, bad) => {
    const s = $(".adm-status");
    if (!s) return;
    s.textContent = msg;
    s.classList.toggle("bad", !!bad);
  };

  opener.addEventListener("click", async () => {
    dirty = false;
    host.innerHTML = `<p class="adm-loading">${d.loading}</p>`;
    dlg.showModal();
    try {
      const r = await fetch("/admin");
      if (!r.ok) throw new Error(r.status);
      host.innerHTML = await r.text();
    } catch {
      host.innerHTML = `<p class="adm-loading bad">${d.failed}</p><p class="adm-loading"><button type="button" class="btn sm" data-close>${d.back}</button></p>`;
    }
  });

  // The look picker is one dialog shared by every card.
  const picker = () => $("#adm-pick");
  const radio = (name, v) => {
    const r = [...picker().querySelectorAll(`input[name="${name}"]`)].find((i) => i.value === v);
    if (r) r.checked = true;
  };
  const chosen = (name) => (picker().querySelector(`input[name="${name}"]:checked`) || {}).value || "";
  const paint = () => ($("#adm-pick-pv").firstElementChild.className = sprite(chosen("shape"), chosen("skin")));

  function openPick(btn) {
    picking = btn.closest(".adm-card");
    const own = picking.querySelector('input[name="shape"]').value;
    const skin = picking.querySelector('input[name="skin"]').value;
    radio("shape", own || btn.dataset.tshape || "jar");
    radio("skin", own ? skin : btn.dataset.tskin || skin);
    picker().querySelector("[data-like]").hidden = !("tshape" in btn.dataset); // only products follow a type
    paint();
    picker().showModal();
  }

  // inherit: hand the look back to the type. Choosing the type's own look counts as that too.
  function endPick(inherit) {
    const btn = picking.querySelector("[data-pick]");
    const t = btn.dataset;
    let shape = chosen("shape"), skin = chosen("skin");
    if (inherit || (t.tshape !== undefined && shape === t.tshape && skin === (t.tskin || ""))) shape = skin = "";
    picking.querySelector('input[name="shape"]').value = shape;
    picking.querySelector('input[name="skin"]').value = skin;
    btn.classList.toggle("own", !!shape);
    btn.querySelector(".it").className = shape ? sprite(shape, skin) : sprite(t.tshape || chosen("shape"), t.tshape ? t.tskin : chosen("skin"));
    picking.classList.add("changed");
    picker().close();
  }

  // A destructive button asks twice: the first press turns it into "¿Seguro?" for 3 s.
  function armed(btn) {
    const disarm = () => {
      clearTimeout(+btn.dataset.armed);
      delete btn.dataset.armed;
      btn.textContent = btn.dataset.label;
    };
    if (btn.dataset.armed) {
      disarm();
      return true;
    }
    btn.dataset.label = btn.textContent;
    btn.textContent = d.sure;
    btn.dataset.armed = setTimeout(disarm, 3000);
    return false;
  }

  dlg.addEventListener("click", (e) => {
    const b = e.target.closest("[data-close],[data-tab],[data-pick],[data-like],[data-cancel],[data-done]");
    if (!b) return;
    if (b.matches("[data-close]")) dlg.close();
    else if (b.matches("[data-pick]")) openPick(b);
    else if (b.matches("[data-like]")) endPick(true);
    else if (b.matches("[data-done]")) endPick(false);
    else if (b.matches("[data-cancel]")) picker().close();
    else {
      dlg.querySelectorAll(".adm-tab").forEach((t) => t.setAttribute("aria-selected", String(t === b)));
      dlg.querySelectorAll(".adm-pane").forEach((p) => (p.hidden = p.dataset.pane !== b.dataset.tab));
    }
  });

  dlg.addEventListener("input", (e) => {
    const t = e.target;
    if (t.closest("#adm-pick")) {
      paint();
    } else if (t.matches(".adm-filter")) {
      const q = fold(t.value.trim());
      dlg.querySelectorAll('[data-pane="products"] .adm-card').forEach((c) => (c.hidden = !!q && !fold(c.dataset.q).includes(q)));
    } else if (t.closest("form[data-adm]:not([data-swap])")) {
      t.closest(".adm-card").classList.add("changed"); // unsaved
    }
  });

  dlg.addEventListener("submit", async (e) => {
    const f = e.target.closest("form[data-adm]");
    if (!f) return;
    e.preventDefault();
    const btn = e.submitter;
    if (f.hasAttribute("data-sure") && !armed(btn)) return;
    btn.disabled = true;
    try {
      const r = await fetch(f.action, { method: "POST", body: new URLSearchParams(new FormData(f)) });
      if (!r.ok) throw new Error((await r.json().catch(() => ({}))).message || d.bad);
      dirty = true;
      const target = f.closest(f.dataset.swap || ".adm-card");
      if (f.hasAttribute("data-gone")) target.remove();
      else target.outerHTML = await r.text();
      say(d.saved);
    } catch (err) {
      say(err.message, true);
      btn.disabled = false;
    }
  });

  dlg.addEventListener("close", () => {
    if (dirty) location.reload();
  });
})();
