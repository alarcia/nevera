// Add dialog: live preview, catalog re-add, suggested expiry and shelf, and
// "save and add another". Data and copy come from the server (add-data,
// data-* attrs); opening and closing are native (commandfor / closedby).
(() => {
  const dlg = document.getElementById("add");
  if (!dlg) return;
  const data = JSON.parse(document.getElementById("add-data").textContent);
  const form = dlg.querySelector("form");
  const $ = (s) => dlg.querySelector(s);
  const f = form.elements;
  const types = Object.fromEntries(data.Types.map((t) => [t.ID, t]));
  const byId = Object.fromEntries((data.Products || []).map((p) => [p.ID, p]));
  const preview = $(".pv .it");
  const nameOut = $(".pv-name");
  const touched = { look: false, size: false, exp: false, comp: false };
  let picked = null; // catalog product being re-added
  let count = 0, lastComp = ""; // saved with "add another" since the dialog opened
  const note = $(".saved-note");

  const val = (name) => (f[name] && f[name].value) || "";
  const plusDays = (n) => {
    const d = new Date(data.Today + "T00:00:00Z");
    d.setUTCDate(d.getUTCDate() + n);
    return d.toISOString().slice(0, 10);
  };
  const fold = (s) => s.normalize("NFD").replace(/\p{M}/gu, "").toLowerCase();
  const restart = (el, cls) => { el.classList.remove(cls); void el.offsetWidth; el.classList.add(cls); };

  function draw(shape, skin) {
    preview.className = `it spin k-${shape || "jar"}` + (skin ? ` c-${skin}` : "");
    restart(preview.closest(".pv"), "pop");
  }

  function setRadio(name, v) {
    const r = [...form.querySelectorAll(`input[name="${name}"]`)].find((i) => i.value === v);
    if (r) r.checked = true;
  }

  // Expiry: today + the product's (or its type's) usual shelf life; 0 = none.
  // The user has the last word: typing a date or ticking "no date" stops it.
  function defaultExpiry() {
    if (touched.exp || f.noExpiry.checked) return;
    const t = types[picked ? picked.Type : val("type")];
    const life = (picked && picked.Life) || (t && t.Life) || 0;
    f.expiresOn.value = life ? plusDays(life) : "";
  }

  function suggest() {
    const s = picked && picked.Comp ? { Comp: picked.Comp, Why: dlg.dataset.sugOwn } : data.Suggest[picked ? picked.Type : val("type")];
    dlg.querySelectorAll(".mm-z.sug").forEach((z) => z.classList.remove("sug"));
    $(".why-sug").textContent = s ? s.Why : "";
    if (!s) return;
    const r = form.querySelector(`input[name="compartment"][value="${s.Comp}"]`);
    if (!r) return;
    const z = r.closest(".mm-z");
    z.classList.add("sug");
    z.querySelector(".mm-tag").textContent = $(".minimap").dataset.suggested;
    if (!touched.comp) r.checked = true;
  }

  function pick(p, focus) {
    picked = p;
    f.product.value = p ? p.ID : "";
    dlg.querySelectorAll(".new").forEach((n) => { n.disabled = n.hidden = !!p; });
    $(".known").hidden = !p;
    $(".sugg").hidden = true;
    if (p) {
      $(".known-name").textContent = p.Name;
      $(".known-brand").textContent = p.Brand;
      $(".known-it").innerHTML = preview.parentElement.innerHTML;
      $(".known-it .it").className = `it k-${p.Shape}` + (p.Skin ? ` c-${p.Skin}` : "");
      nameOut.textContent = p.Brand ? `${p.Name} · ${p.Brand}` : p.Name;
      draw(p.Shape, p.Skin);
      f.opened.checked = !!(types[p.Type] && types[p.Type].StartsOpen);
    } else {
      nameOut.textContent = f.name.value || dlg.dataset.new;
      draw(val("shape"), val("skin"));
      if (focus) f.name.focus();
    }
    defaultExpiry();
    suggest();
  }

  // Catalog matches while typing a name: one tap re-adds a known product.
  function matches() {
    const q = fold(f.name.value.trim());
    const list = $(".sugg");
    list.replaceChildren();
    if (!q) { list.hidden = true; return; }
    const hits = Object.values(byId).filter((p) => fold(p.Name + " " + p.Brand).includes(q)).slice(0, 5);
    for (const p of hits) {
      const li = document.createElement("li");
      const b = document.createElement("button");
      b.type = "button";
      b.className = "sugg-it";
      b.innerHTML = `<span class="mini">${preview.outerHTML}</span><b></b><span></span>`;
      b.querySelector(".it").className = `it k-${p.Shape}` + (p.Skin ? ` c-${p.Skin}` : "");
      b.querySelector("b").textContent = p.Name;
      b.querySelector("b + span").textContent = p.Brand;
      b.addEventListener("click", () => pick(p));
      li.append(b);
      list.append(li);
    }
    list.hidden = !hits.length;
  }

  form.addEventListener("input", (e) => {
    const n = e.target.name;
    if (n === "name") {
      nameOut.textContent = f.name.value.trim() || dlg.dataset.new;
      matches();
    } else if (n === "type") {
      const t = types[val("type")];
      f.opened.checked = !!(t && t.StartsOpen);
      if (!touched.look && t && t.Shape) {
        setRadio("shape", t.Shape);
        setRadio("skin", t.Skin);
        draw(t.Shape, t.Skin);
      }
      if (!touched.size && t && t.Size) setRadio("size", t.Size);
      defaultExpiry();
      suggest();
    } else if (n === "size") {
      touched.size = true;
    } else if (n === "shape" || n === "skin") {
      touched.look = true;
      draw(val("shape"), val("skin"));
    } else if (n === "expiresOn") {
      touched.exp = true;
    } else if (n === "noExpiry") {
      f.expiresOn.disabled = f.noExpiry.checked;
      if (f.noExpiry.checked) f.expiresOn.value = "";
      else { touched.exp = false; defaultExpiry(); }
    } else if (n === "compartment") {
      touched.comp = true;
    }
  });

  $(".known-change").addEventListener("click", () => pick(null, true));

  const submits = form.querySelectorAll('[type="submit"]');
  const busy = (on) => submits.forEach((b) => (b.disabled = on));
  // Look and size are the type's unless the user chose them: leave them out of the post.
  const own = () => form.querySelectorAll("[data-own]").forEach((g) => (g.disabled = !touched[g.dataset.own]));

  // "Save and add another": post in the background, keep the dialog open and
  // start over, so a bag of groceries goes in without leaving the form.
  // Without JS the button is a plain save.
  async function saveAnother() {
    const comp = form.querySelector('input[name="compartment"]:checked');
    const label = [picked ? picked.Name : f.name.value.trim(), comp.parentElement.querySelector(".mm-n").textContent];
    busy(true);
    try {
      const r = await fetch(form.action, { method: "POST", body: new URLSearchParams(new FormData(form)), redirect: "manual" });
      if (!r.ok && r.type !== "opaqueredirect") throw new Error(r.status);
      count++;
      lastComp = comp.value;
      fresh();
      note.textContent = dlg.dataset.savedNote.replace("{0}", label[0]).replace("{1}", label[1]).replace("{2}", count);
      f.name.focus();
    } catch {
      note.textContent = dlg.dataset.bad;
    }
    note.hidden = false;
    busy(false);
  }

  form.addEventListener("submit", (e) => {
    own();
    if (e.submitter && e.submitter.name === "again") {
      e.preventDefault();
      saveAnother();
    } else {
      setTimeout(() => busy(true));
    }
  });

  // A fresh, blank form every time the dialog opens or a product was saved.
  function fresh() {
    form.reset();
    form.querySelectorAll("[data-own]").forEach((g) => (g.disabled = false));
    f.expiresOn.disabled = false;
    $(".extras").open = false;
    Object.keys(touched).forEach((k) => (touched[k] = false));
    pick(null);
  }
  fresh();
  dlg.addEventListener("toggle", (e) => {
    if (e.newState !== "open") return;
    count = 0;
    note.hidden = true;
    busy(false);
    fresh();
  });
  // Closing after "add another": reload so the fridge shows what went in.
  dlg.addEventListener("close", () => {
    if (count) location.href = "/?added=" + encodeURIComponent(lastComp);
  });
  // Tap on the backdrop closes it (closedby="any" does it natively where supported).
  dlg.addEventListener("click", (e) => e.target === dlg && dlg.close());
  // Browsers without invoker commands (commandfor / command).
  if (!("command" in HTMLButtonElement.prototype)) {
    document.addEventListener("click", (e) => {
      const b = e.target.closest("[commandfor]");
      if (!b) return;
      if (b.getAttribute("command") === "show-modal") dlg.showModal();
      else dlg.close();
    });
  }

  // Back from a save: flash the compartment and drop the new item in.
  const added = new URLSearchParams(location.search).get("added");
  if (added) {
    history.replaceState(null, "", location.pathname);
    const zone = document.querySelector(`.zone[href="/c/${CSS.escape(added)}"]`);
    const toast = document.querySelector(".toast");
    if (zone) {
      zone.classList.add("just-added");
      if (zone.classList.contains("door")) {
        const t = document.getElementById("door-toggle");
        if (t) t.checked = true;
      }
      toast.textContent = dlg.dataset.savedIn + " " + zone.querySelector(".zname").textContent;
      toast.classList.add("show");
      setTimeout(() => toast.classList.remove("show"), 2600);
    }
  }
})();
