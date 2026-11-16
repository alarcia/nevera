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
  const $$ = (s, root = dlg) => [...root.querySelectorAll(s)];
  const fold = (s) => s.normalize("NFD").replace(/\p{M}/gu, "").toLowerCase();
  const sprite = (shape, skin) => `it k-${shape || "jar"}` + (skin ? ` c-${skin}` : "");
  const say = (msg, bad) => {
    const s = $(".adm-status");
    if (!s) return;
    s.textContent = msg;
    s.classList.toggle("bad", !!bad);
  };

  // A card is "changed" (yellow) only while its form differs from how it was drawn:
  // change something and put it back, and there is nothing left to save.
  const state = (card) => new URLSearchParams(new FormData(card.querySelector("form[data-adm]:not([data-swap])"))).toString();
  const baseline = (cards) => cards.forEach((c) => (c.dataset.snap = state(c)));
  const check = (card) => card.classList.toggle("changed", state(card) !== card.dataset.snap);

  // Escape must never close it: one stray key press would throw the edits away.
  // (closedby="none" does it where supported; this covers the rest.)
  dlg.addEventListener("cancel", (e) => e.preventDefault());

  opener.addEventListener("click", async () => {
    dirty = false;
    host.innerHTML = `<p class="adm-loading">${d.loading}</p>`;
    dlg.showModal();
    try {
      const r = await fetch("/admin");
      if (!r.ok) throw new Error(r.status);
      host.innerHTML = await r.text();
      baseline($$(".adm-card"));
    } catch {
      host.innerHTML = `<p class="adm-loading bad">${d.failed}</p><p class="adm-loading"><button type="button" class="btn sm" data-close>${d.back}</button></p>`;
    }
  });

  // The look picker is one dialog shared by every card.
  const picker = () => $("#adm-pick");
  const radio = (name, v) => {
    const r = $$(`input[name="${name}"]`, picker()).find((i) => i.value === v);
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
    if (t.tshape !== undefined) btn.classList.toggle("own", !!shape); // only a product can differ from its type
    btn.querySelector(".it").className = shape ? sprite(shape, skin) : sprite(t.tshape || chosen("shape"), t.tshape ? t.tskin : chosen("skin"));
    check(picking);
    picker().close();
  }

  // A destructive button asks twice: the first press turns it into "¿Seguro?" for 3 s.
  // A second press right away (a double click) is not an answer.
  function armed(btn, text = d.sure) {
    const disarm = () => {
      clearTimeout(+btn.dataset.armed);
      delete btn.dataset.armed;
      btn.textContent = btn.dataset.label;
    };
    if (btn.dataset.armed) {
      if (Date.now() - btn.dataset.at < 400) return false;
      disarm();
      return true;
    }
    btn.dataset.at = Date.now();
    btn.dataset.label = btn.textContent;
    btn.textContent = text;
    btn.dataset.armed = setTimeout(disarm, 3000);
    return false;
  }

  // A product leaves some things to its category. Keep what that means on screen
  // (the look it inherits, "like the category (Medium)", the days hint) in step
  // with the category chosen in its list.
  function syncType(card) {
    const btn = card.querySelector("[data-pick]");
    if (!btn || !("tshape" in btn.dataset)) return; // category cards have nothing to follow
    const t = card.querySelector('input[name="type"]:checked');
    btn.dataset.tshape = (t && t.dataset.shape) || "jar";
    btn.dataset.tskin = (t && t.dataset.skin) || "";
    if (!card.querySelector('input[name="shape"]').value) btn.querySelector(".it").className = sprite(btn.dataset.tshape, btn.dataset.tskin);
    $$("input[data-from]", card).forEach((i) => {
      const v = t ? t.dataset[i.dataset.from] : "";
      const label = i.nextElementSibling;
      label.textContent = v ? `${d.liketype} (${v})` : d.liketype;
      if (i.checked) i.closest(".dd").querySelector(".dd-val").textContent = label.textContent;
    });
    card.querySelector('input[name="shelfLifeDays"]').placeholder = t ? `${d.liketype} (${t.dataset.life})` : d.liketype;
  }

  // A category was created or saved: bring its entry up to date in every product's list.
  function syncOpts(root) {
    const tpl = root.querySelector("template.adm-opt");
    if (!tpl) return;
    const fresh = tpl.content.querySelector(".dd-opt");
    const id = fresh.querySelector("input").value;
    $$('.dd[data-dd="type"] .dd-list').forEach((list) => {
      const mine = fresh.cloneNode(true);
      const old = $$("input", list).find((i) => i.value === id);
      if (old) {
        mine.querySelector("input").checked = old.checked;
        if (old.checked) list.closest(".dd").querySelector(".dd-val").textContent = mine.textContent;
        old.closest(".dd-opt").replaceWith(mine);
      } else {
        list.append(mine);
      }
    });
    $$('[data-pane="products"] .adm-card').forEach(syncType);
  }

  // The days stepper: empty is "not set", then 0 (the same day), 1, 2…
  function step(btn) {
    const i = btn.parentElement.querySelector("input");
    const v = i.value === "" ? null : +i.value;
    i.value = btn.dataset.step > 0 ? (v === null ? 0 : Math.min(+i.dataset.max, v + 1)) : v ? v - 1 : "";
    i.dispatchEvent(new Event("input", { bubbles: true }));
  }

  const closeDd = (dd, focus) => {
    dd.open = false;
    if (focus) dd.querySelector("summary").focus();
  };

  dlg.addEventListener("click", (e) => {
    const here = e.target.closest(".dd");
    $$(".dd[open]").forEach((o) => o !== here && closeDd(o));
    const opt = e.target.closest(".dd-opt");
    if (opt && e.detail) closeDd(here); // a real click or tap picks and closes; arrow keys keep it open
    const b = e.target.closest("[data-close],[data-tab],[data-pick],[data-like],[data-cancel],[data-done],[data-step]");
    if (!b) return;
    if (b.matches("[data-close]")) {
      if ($(".adm-card.changed") && !armed(b, d.unsaved)) return; // edits not saved yet
      dlg.close();
    } else if (b.matches("[data-step]")) step(b);
    else if (b.matches("[data-pick]")) openPick(b);
    else if (b.matches("[data-like]")) endPick(true);
    else if (b.matches("[data-done]")) endPick(false);
    else if (b.matches("[data-cancel]")) picker().close();
    else {
      $$(".adm-tab").forEach((t) => t.setAttribute("aria-selected", String(t === b)));
      $$(".adm-pane").forEach((p) => (p.hidden = p.dataset.pane !== b.dataset.tab));
    }
  });

  // Enter on an option closes the list (it would otherwise submit the card); Escape closes just the list.
  dlg.addEventListener("keydown", (e) => {
    const dd = e.target.closest(".dd");
    if (dd && (e.key === "Escape" || (e.key === "Enter" && e.target.closest(".dd-list")))) {
      e.preventDefault();
      closeDd(dd, true);
    }
  });

  // An opened list scrolls into view instead of hiding under the screen's edge. (toggle does not bubble.)
  dlg.addEventListener("toggle", (e) => e.target.open && e.target.matches(".dd") && e.target.querySelector(".dd-list").scrollIntoView({ block: "nearest" }), true);

  dlg.addEventListener("change", (e) => {
    const t = e.target;
    if (!t.matches(".dd-opt input")) return;
    const val = t.closest(".dd").querySelector(".dd-val");
    val.textContent = t.nextElementSibling.textContent;
    val.classList.remove("dd-ph");
    if (t.name === "type") syncType(t.closest(".adm-card"));
  });

  dlg.addEventListener("input", (e) => {
    const t = e.target;
    if (t.closest("#adm-pick")) {
      paint();
      return;
    }
    if (t.matches(".adm-num input")) t.value = t.value.replace(/\D/g, "");
    if (t.matches(".adm-filter")) {
      const q = fold(t.value.trim()); // both tabs: a category and a product can share a name
      $$(".adm-card[data-q]").forEach((c) => (c.hidden = !!q && !fold(c.dataset.q).includes(q)));
    } else if (t.closest("form[data-adm]:not([data-swap])")) {
      check(t.closest(".adm-card"));
    }
  });

  dlg.addEventListener("submit", async (e) => {
    const f = e.target.closest("form[data-adm]");
    if (!f) return;
    e.preventDefault();
    const btn = e.submitter;
    if (f.hasAttribute("data-sure") && !armed(btn)) return;
    const type = f.querySelector('.dd[data-dd="type"]');
    if (type && !f.querySelector('input[name="type"]:checked')) {
      say(d.picktype, true); // a new product needs its category
      type.open = true;
      return;
    }
    btn.disabled = true;
    try {
      const r = await fetch(f.action, { method: "POST", body: new URLSearchParams(new FormData(f)) });
      if (!r.ok) throw new Error((await r.json().catch(() => ({}))).message || d.bad);
      dirty = true;
      const target = f.closest(f.dataset.swap || ".adm-card");
      if (f.hasAttribute("data-gone")) {
        target.remove();
      } else {
        const page = document.createElement("template");
        page.innerHTML = await r.text();
        syncOpts(page.content);
        const cards = $$(".adm-card", page.content);
        target.replaceWith(page.content);
        baseline(cards);
      }
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
