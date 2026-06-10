// ----------------------------------------------------------------
// Officetracker ledger
// State ints (API contract): 0 untracked · 1 home · 2 office ·
// 3 other · 4/5/6 scheduled (planned) variants of 1/2/3.
// The tracking year runs October–September: GET /api/v1/state/{y}
// returns months 1–12 where months 10–12 belong to calendar y-1.
// ----------------------------------------------------------------
"use strict";

var MONTHS = ["January", "February", "March", "April", "May", "June", "July",
    "August", "September", "October", "November", "December"];
var WEEKDAYS = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];
var STAMP_LABEL = ["", "Home", "Office", "Other", "Home", "Office", "Other"];
var STATE_NAME = ["Not tracked", "Worked from home", "In the office", "Other",
    "Planned: home", "Planned: office", "Planned: other"];

var rawState = {{ .YearlyState }};
var rawNotes = {{ .YearlyNotes }};

var calendarEl = document.getElementById("calendar");
var monthNameEl = document.getElementById("month-name");
var monthYrEl = document.getElementById("month-yr");
var notesEl = document.getElementById("notes");
var notesMonthEl = document.getElementById("notes-month");
var summaryTableEl = document.getElementById("summary-table");
var statPctEl = document.getElementById("stat-pct");
var statSubEl = document.getElementById("stat-sub");
var statBarEl = document.getElementById("stat-bar-fill");
var statRangeEl = document.getElementById("stat-range");

var state = {};   // { month(1-12): { day: stateInt } }
var notes = {};   // { month(1-12): "..." }
var currentYear;  // calendar year of the visible month
var currentMonth; // 0-based visible month

// ---------------------------------------------------------------- helpers

function fiscalYearOf(year, monthIdx) {
    // months Oct–Dec belong to the following tracking year
    return monthIdx < 9 ? year : year + 1;
}

function pad2(n) { return (n < 10 ? "0" : "") + n; }

function mapState(payload) {
    var out = {};
    var months = (payload && payload.data && payload.data.months) || {};
    for (var m in months) {
        out[m] = {};
        var days = months[m].days || {};
        for (var d in days) {
            out[m][d] = days[d].state;
        }
    }
    return out;
}

function mapNotes(payload) {
    var out = {};
    var data = (payload && payload.data) || {};
    for (var m in data) {
        out[m] = data[m].note;
    }
    return out;
}

function stateOf(day) {
    var m = state[currentMonth + 1];
    return (m && day in m) ? m[day] : 0;
}

function parseLocation() {
    var m = window.location.pathname.match(/(\d{4})-(\d{1,2})$/);
    if (m) {
        currentYear = parseInt(m[1], 10);
        currentMonth = parseInt(m[2], 10) - 1;
    } else {
        var now = new Date();
        currentYear = now.getFullYear();
        currentMonth = now.getMonth();
    }
}

// ---------------------------------------------------------------- data

function refetchAll() {
    refetchState();
    refetchNotes();
}

function refetchState() {
    var fy = fiscalYearOf(currentYear, currentMonth);
    fetch("/api/v1/state/" + fy)
        .then(function (r) { return r.json(); })
        .then(function (payload) {
            state = mapState(payload);
            renderCalendar();
            renderSummary();
        })
        .catch(function () { OT.error("Could not load the year"); });
}

function refetchNotes() {
    var fy = fiscalYearOf(currentYear, currentMonth);
    fetch("/api/v1/note/" + fy)
        .then(function (r) { return r.json(); })
        .then(function (payload) {
            notes = mapNotes(payload);
            renderNotes();
        })
        .catch(function () { /* notes are non-critical */ });
}

// Rapid clicks cycle through several states; debounce so only the final
// state is written, avoiding concurrent PUTs racing each other.
var pendingSaves = {}; // "y-m-d" -> { y, m, d, state, timer }

function persistDay(day, st) {
    var m = currentMonth + 1;
    if (!(m in state)) state[m] = {};
    state[m][day] = st;

    var key = currentYear + "-" + m + "-" + day;
    if (pendingSaves[key]) clearTimeout(pendingSaves[key].timer);
    pendingSaves[key] = {
        y: currentYear, m: m, d: day, state: st,
        timer: setTimeout(function () { flushDaySave(key); }, 300)
    };
}

function flushDaySave(key) {
    var p = pendingSaves[key];
    if (!p) return;
    delete pendingSaves[key];
    clearTimeout(p.timer);

    fetch("/api/v1/state/" + p.y + "/" + p.m + "/" + p.d, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ data: { state: p.state } }),
        credentials: "include",
        keepalive: true
    }).then(function (r) {
        if (!r.ok) throw new Error("save failed");
        OT.saved();
        if (p.state === 0) {
            // clearing a day may fall back to a scheduled (planned) state
            setTimeout(refetchState, 150);
        } else {
            renderSummary();
        }
    }).catch(function () {
        OT.error("Day not recorded — try again");
    });
}

window.addEventListener("pagehide", function () {
    for (var key in pendingSaves) flushDaySave(key);
});

function saveNote() {
    var val = notesEl.value;
    if ((notes[currentMonth + 1] || "") === val) return;
    notes[currentMonth + 1] = val;

    fetch("/api/v1/note/" + currentYear + "/" + (currentMonth + 1), {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ data: { note: val } }),
        credentials: "include"
    }).then(function (r) {
        if (!r.ok) throw new Error("save failed");
        OT.saved();
    }).catch(function () {
        OT.error("Note not saved — try again");
    });
}

// ---------------------------------------------------------------- calendar

function ariaFor(day, st) {
    var wd = WEEKDAYS[new Date(currentYear, currentMonth, day).getDay()];
    return wd + " " + day + " " + MONTHS[currentMonth] + ": " + STATE_NAME[st] + ". Press to change.";
}

function makeStamp(st) {
    var stamp = document.createElement("span");
    stamp.className = "dstamp";
    stamp.textContent = STAMP_LABEL[st];
    return stamp;
}

function applyDayState(btn, st) {
    btn.dataset.state = st;
    var old = btn.querySelector(".dstamp");
    if (old) old.remove();
    btn.appendChild(makeStamp(st));
    btn.setAttribute("aria-label", ariaFor(parseInt(btn.dataset.day, 10), st));
}

function cycleDay(btn, dir) {
    var orig = parseInt(btn.dataset.state, 10);
    var base = orig >= 4 ? 0 : orig; // planned states reset to untracked before cycling
    var next = (base + dir + 4) % 4;
    applyDayState(btn, next);
    persistDay(parseInt(btn.dataset.day, 10), next);
}

function renderCalendar() {
    var grid = document.createElement("div");
    grid.className = "cal-grid";

    ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"].forEach(function (d) {
        var wd = document.createElement("span");
        wd.className = "cal-wd";
        wd.textContent = d;
        grid.appendChild(wd);
    });

    var lead = (new Date(currentYear, currentMonth, 1).getDay() || 7) - 1;
    var daysInMonth = new Date(currentYear, currentMonth + 1, 0).getDate();
    var today = new Date();

    function blank() {
        var b = document.createElement("span");
        b.className = "day-blank";
        b.setAttribute("aria-hidden", "true");
        grid.appendChild(b);
    }

    for (var i = 0; i < lead; i++) blank();

    for (var d = 1; d <= daysInMonth; d++) {
        var st = stateOf(d);
        var btn = document.createElement("button");
        btn.type = "button";
        btn.className = "day";
        btn.dataset.day = d;
        btn.dataset.state = st;

        var dow = new Date(currentYear, currentMonth, d).getDay();
        if (dow === 0 || dow === 6) btn.classList.add("weekend");
        if (today.getFullYear() === currentYear && today.getMonth() === currentMonth && today.getDate() === d) {
            btn.classList.add("today");
        }

        var num = document.createElement("span");
        num.className = "dnum";
        num.textContent = d;
        btn.appendChild(num);

        btn.appendChild(makeStamp(st));

        btn.setAttribute("aria-label", ariaFor(d, st));

        btn.addEventListener("click", function () { cycleDay(this, 1); });
        btn.addEventListener("contextmenu", function (e) {
            e.preventDefault();
            cycleDay(this, -1);
        });
        btn.addEventListener("mouseenter", function (e) {
            showTooltip(e, parseInt(this.dataset.day, 10));
        });
        btn.addEventListener("mouseleave", hideTooltip);

        grid.appendChild(btn);
    }

    var filled = lead + daysInMonth;
    var trail = (7 - (filled % 7)) % 7;
    for (i = 0; i < trail; i++) blank();

    calendarEl.innerHTML = "";
    calendarEl.appendChild(grid);
}

// ---------------------------------------------------------------- tooltip (running totals)

function monthToDate(upToDay) {
    var present = 0, total = 0;
    var m = state[currentMonth + 1] || {};
    for (var d = 1; d <= upToDay; d++) {
        var st = m[d];
        if (st === 2 || st === 5) { present++; total++; }
        else if (st === 1 || st === 4) { total++; }
    }
    return { present: present, total: total };
}

function yearToDate(upToDay) {
    var present = 0, total = 0;
    var months = Object.keys(state).map(function (m) { return parseInt(m, 10); });
    months.sort(function (a, b) {
        // tracking-year order: Oct, Nov, Dec, Jan … Sep
        var ao = a >= 10 ? a - 10 : a + 2;
        var bo = b >= 10 ? b - 10 : b + 2;
        return ao - bo;
    });

    for (var i = 0; i < months.length; i++) {
        var m = months[i];
        var isCurrent = (m === currentMonth + 1);
        var days = state[m];
        for (var d in days) {
            if (isCurrent && parseInt(d, 10) > upToDay) continue;
            var st = days[d];
            if (st === 2 || st === 5) { present++; total++; }
            else if (st === 1 || st === 4) { total++; }
        }
        if (isCurrent) break;
    }
    return { present: present, total: total };
}

function pct(present, total) {
    return total > 0 ? ((present / total) * 100).toFixed(1) : "0.0";
}

function showTooltip(event, day) {
    hideTooltip();

    var mtd = monthToDate(day);
    var ytd = yearToDate(day);

    var tip = document.createElement("div");
    tip.className = "day-tooltip";
    tip.id = "calendar-tooltip";

    var head = document.createElement("strong");
    head.textContent = "Through " + MONTHS[currentMonth] + " " + day;
    tip.appendChild(head);
    tip.appendChild(document.createElement("br"));
    tip.appendChild(document.createTextNode(
        "Month " + mtd.present + "/" + mtd.total + " (" + pct(mtd.present, mtd.total) + "%)"));
    tip.appendChild(document.createElement("br"));
    tip.appendChild(document.createTextNode(
        "Year " + ytd.present + "/" + ytd.total + " (" + pct(ytd.present, ytd.total) + "%)"));

    document.body.appendChild(tip);

    var rect = event.target.getBoundingClientRect();
    var tipRect = tip.getBoundingClientRect();
    tip.style.left = (rect.left + rect.width / 2 - tipRect.width / 2 + window.scrollX) + "px";
    tip.style.top = (rect.top - tipRect.height - 10 + window.scrollY) + "px";
}

function hideTooltip() {
    var existing = document.getElementById("calendar-tooltip");
    if (existing) existing.remove();
}

// ---------------------------------------------------------------- summary

function renderSummary() {
    var fy = fiscalYearOf(currentYear, currentMonth);
    var order = [10, 11, 12, 1, 2, 3, 4, 5, 6, 7, 8, 9];
    var allPresent = 0, allTotal = 0;

    var thead = document.createElement("thead");
    var hrow = document.createElement("tr");
    [["Month", ""], ["In", "num"], ["Days", "num"], ["Rate", "num"]].forEach(function (h) {
        var th = document.createElement("th");
        th.textContent = h[0];
        if (h[1]) th.className = h[1];
        hrow.appendChild(th);
    });
    thead.appendChild(hrow);

    var tbody = document.createElement("tbody");

    order.forEach(function (m) {
        var days = state[m];
        if (!days) return;

        var counts = [0, 0, 0, 0, 0, 0, 0];
        for (var d in days) counts[days[d]]++;

        var present = counts[2] + counts[5];
        var total = counts[1] + counts[2] + counts[4] + counts[5];
        allPresent += present;
        allTotal += total;
        if (total <= 0) return;

        var calYear = m >= 10 ? fy - 1 : fy;
        var tr = document.createElement("tr");
        if (m === currentMonth + 1) tr.className = "now";

        var cells = [
            [MONTHS[m - 1].slice(0, 3) + " " + calYear, ""],
            [String(present), "num"],
            [String(total), "num"],
            [pct(present, total) + "%", "num"]
        ];
        cells.forEach(function (c) {
            var td = document.createElement("td");
            td.textContent = c[0];
            if (c[1]) td.className = c[1];
            tr.appendChild(td);
        });
        tbody.appendChild(tr);
    });

    if (!tbody.children.length) {
        var tr = document.createElement("tr");
        var td = document.createElement("td");
        td.colSpan = 4;
        td.className = "empty";
        td.textContent = "Nothing stamped yet.";
        tr.appendChild(td);
        tbody.appendChild(tr);
    }

    summaryTableEl.innerHTML = "";
    summaryTableEl.appendChild(thead);
    summaryTableEl.appendChild(tbody);

    // headline
    if (allTotal > 0) {
        var rate = (allPresent / allTotal) * 100;
        statPctEl.textContent = rate.toFixed(1) + "%";
        statSubEl.textContent = allPresent + " of " + allTotal + " working days in the office";
        statBarEl.style.width = Math.min(rate, 100) + "%";
    } else {
        statPctEl.textContent = "—";
        statSubEl.textContent = "No working days recorded yet";
        statBarEl.style.width = "0%";
    }
    statRangeEl.textContent = "Oct " + (fy - 1) + " — Sep " + fy;
}

// ---------------------------------------------------------------- notes & title

function renderNotes() {
    notesEl.value = notes[currentMonth + 1] || "";
    notesMonthEl.textContent = MONTHS[currentMonth];
}

function renderTitle() {
    monthNameEl.textContent = MONTHS[currentMonth];
    monthYrEl.textContent = String(currentYear);
}

function renderAll() {
    renderTitle();
    renderCalendar();
    renderNotes();
    renderSummary();
}

// ---------------------------------------------------------------- navigation

function afterNav(fyBefore) {
    renderTitle();
    if (fiscalYearOf(currentYear, currentMonth) !== fyBefore) {
        refetchAll();
    } else {
        renderAll();
    }
}

function gotoMonth(delta) {
    var fyBefore = fiscalYearOf(currentYear, currentMonth);
    currentMonth += delta;
    if (currentMonth < 0) { currentMonth = 11; currentYear--; }
    else if (currentMonth > 11) { currentMonth = 0; currentYear++; }
    window.history.pushState({}, "", "/" + currentYear + "-" + pad2(currentMonth + 1));
    afterNav(fyBefore);
}

function gotoToday() {
    var fyBefore = fiscalYearOf(currentYear, currentMonth);
    var now = new Date();
    currentYear = now.getFullYear();
    currentMonth = now.getMonth();
    window.history.pushState({}, "", "/" + currentYear + "-" + pad2(currentMonth + 1));
    afterNav(fyBefore);
}

document.getElementById("prev-month").addEventListener("click", function () { gotoMonth(-1); });
document.getElementById("next-month").addEventListener("click", function () { gotoMonth(1); });
document.getElementById("today-btn").addEventListener("click", gotoToday);

window.addEventListener("popstate", function () {
    var fyBefore = fiscalYearOf(currentYear, currentMonth);
    parseLocation();
    afterNav(fyBefore);
});

document.addEventListener("keydown", function (e) {
    if (e.target.closest("input, textarea, select")) return;
    if (e.key === "ArrowLeft") gotoMonth(-1);
    else if (e.key === "ArrowRight") gotoMonth(1);
});

// ---------------------------------------------------------------- notes & export

notesEl.addEventListener("blur", saveNote);

document.getElementById("export-csv").addEventListener("click", function () {
    var fy = fiscalYearOf(currentYear, currentMonth);
    window.location.href = "/api/v1/report/csv/" + fy + "-attendance";
});

var pdfRow = document.getElementById("pdf-name-row");
document.getElementById("export-pdf").addEventListener("click", function () {
    pdfRow.hidden = !pdfRow.hidden;
    if (!pdfRow.hidden) document.getElementById("pdf-name").focus();
});
document.getElementById("pdf-go").addEventListener("click", function () {
    var fy = fiscalYearOf(currentYear, currentMonth);
    var name = document.getElementById("pdf-name").value.trim();
    window.location.href = "/api/v1/report/pdf/" + fy + "-attendance?name=" + encodeURIComponent(name);
});

// ---------------------------------------------------------------- init

parseLocation();
state = mapState(rawState);
notes = mapNotes(rawNotes);
renderAll();
