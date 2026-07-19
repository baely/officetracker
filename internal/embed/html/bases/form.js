const states = ["untracked", "present", "not present", "other", "scheduled-present", "scheduled-not-present", "scheduled-other"];
const monthNames = ["January", "February", "March", "April", "May", "June", "July", "August", "September",
    "October", "November", "December"];

// The month (1-12) the tracking year starts on, configurable per user.
let trackingStartMonth = {{ .TrackingStartMonth }} || 10;

// The user's monthly attendance target percentage (0 = no target set).
let targetPercent = {{ .TargetPercent }} || 0;

// trackingYearForMonth0 maps a 0-indexed calendar month + calendar year to its
// tracking-year label (mirrors util.TrackingYear in Go).
function trackingYearForMonth0(month0, calYear) {
    if (trackingStartMonth === 1) { return calYear; }
    return month0 < (trackingStartMonth - 1) ? calYear : calYear + 1;
}

// trackingMonthOrder returns the position (0-11) of a 1-indexed month within the
// tracking year.
function trackingMonthOrder(month1) {
    return (month1 - trackingStartMonth + 12) % 12;
}

class Data {
    static titleDOM = document.getElementById("month-year");
    static calendarDOM = document.getElementById("calendar");
    static notesDOM = document.getElementById("notes");
    static targetDOM = document.getElementById("target-progress");

    constructor(state, notes) {
        this.state = state;
        this.notes = notes;
        this.updateDate(true, false);
        this.refreshDOM();
        Data.notesDOM.addEventListener("blur", () => { this.updateNote() });
        document.getElementById("prev-month").addEventListener("click", () => this.updateMonth(-1));
        document.getElementById("next-month").addEventListener("click", () => this.updateMonth(1));
        window.addEventListener("popstate", this.updateDate);
    }

    cycleState(dayDOM, direction) {
        let originalState = parseInt(dayDOM.dataset.state);
        let previousState = originalState;
        
        // If this is a scheduled state, convert it to untracked first
        if (previousState >= 4) { // Scheduled states are 4, 5, 6
            previousState = 0; // Convert to untracked
        }
        
        // Only cycle through the first 4 states (0-3), not scheduled states
        let currentState = (previousState + direction + 4) % 4;
        let date = dayDOM.textContent;
        
        // Remove all state classes (use original state for correct class removal)
        dayDOM.classList.remove(getClassForState(states[originalState]));
        
        dayDOM.classList.add(getClassForState(states[currentState]));
        dayDOM.dataset.state = currentState;
        this.updateState(date, currentState);
    }

    drawCalendar() {
        let calendarDOM = generateCalendar(this.currentMonth, this.currentYear, this.state,
            (dayDOM, direction) => this.cycleState(dayDOM, direction)
        );
        Data.calendarDOM.removeAttribute("id");
        calendarDOM.id = "calendar";
        Data.calendarDOM.replaceWith(calendarDOM);
        Data.calendarDOM = calendarDOM;
    }

    drawNotes() {
        if (!(this.currentMonth+1 in this.notes)) {
            this.notes[this.currentMonth+1] = "";
        }
        Data.notesDOM.value = this.notes[this.currentMonth+1];
    }

    // drawTarget renders the monthly attendance target: progress so far this
    // month, how many more office days are needed to meet the target, and an
    // inline picker to set/adjust the target; saving updates the projection
    // immediately.
    drawTarget() {
        const elem = Data.targetDOM;
        let status;
        if (targetPercent <= 0) {
            status = "No monthly attendance target set.";
        } else {
            status = this.targetStatus();
        }

        elem.innerHTML = status + "<br>" +
            'Target: <input type="number" id="target-inline" min="0" max="100" step="10"> % ' +
            '<button id="target-inline-save">Save</button>';
        const input = document.getElementById("target-inline");
        if (targetPercent > 0) { input.value = targetPercent; }
        const save = () => {
            let value = parseInt(input.value, 10);
            if (isNaN(value) || value < 0) { value = 0; } // 0 = no target
            if (value > 100) { value = 100; }
            fetch("/api/v1/settings/target", {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    data: {
                        default_target_percent: value
                    }
                }),
                credentials: "include"
            });
            targetPercent = value;
            this.drawTarget();
        };
        document.getElementById("target-inline-save").addEventListener("click", save);
        input.addEventListener("keydown", (event) => { if (event.key === "Enter") { save(); } });
    }

    // targetStatus builds the progress sentences for the current month against
    // the set target.
    targetStatus() {
        // Count this month's work days the same way the yearly report does:
        // present = office (actual + scheduled), total = WFH + office.
        const days = this.state[this.currentMonth + 1] || {};
        let present = 0, total = 0;
        for (const day in days) {
            const state = days[day];
            if (state === 2 || state === 5) {
                present++;
                total++;
            } else if (state === 1 || state === 4) {
                total++;
            }
        }

        // Project the month-end total: work days tracked so far plus remaining
        // weekdays that aren't tracked yet (assume they will be work days).
        let projectedTotal = total;
        const now = new Date();
        const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate());
        const daysInMonth = new Date(this.currentYear, this.currentMonth + 1, 0).getDate();
        for (let day = 1; day <= daysInMonth; day++) {
            if ((days[day] || 0) !== 0) { continue; } // already tracked
            const date = new Date(this.currentYear, this.currentMonth, day);
            if (date < startOfToday) { continue; } // past untracked days don't count
            const dow = date.getDay();
            if (dow === 0 || dow === 6) { continue; } // weekends
            projectedTotal++;
        }

        const percentage = total > 0 ? ((present / total) * 100).toFixed(1) : "0.0";
        const needed = Math.max(0, Math.ceil(targetPercent / 100 * projectedTotal) - present);
        const neededLine = needed > 0
            ? `<span class="num">${needed}</span> more office day${needed === 1 ? "" : "s"} needed this month.`
            : "Target met for this month.";
        const progressLine = `In office <span class="num">${present}</span> of <span class="num">${total}</span> days (<span class="num">${percentage}%</span>).`;
        return neededLine + "<br>" + progressLine;
    }

    fetchData() {
        let year = trackingYearForMonth0(this.currentMonth, this.currentYear);
        fetch("/api/v1/state/" + year)
            .then(r => r.json())
            .then(payload => {
                this.state = mapState(payload);
                this.refreshDOM();
            });
    }

    fetchNotes() {
        let year = trackingYearForMonth0(this.currentMonth, this.currentYear);
        fetch("/api/v1/note/" + year)
            .then(r => r.json())
            .then(payload => {
                this.notes = mapNotes(payload);
                this.refreshDOM();
            });
    }

    refreshDOM() {
        this.drawCalendar();
        this.drawNotes();
        this.drawTarget();
        this.updateTitle();
    }

    updateBackend(day) {
        let month = "" + (this.currentMonth + 1);
        let year = "" + this.currentYear;
        if (!(month in this.state)) {
            this.state[month] = {};
        }
        let thisState = this.state[month][day];

        let obj = {
            "data": {
                "state": thisState
            }
        };
        fetch("/api/v1/state/" + year + "/" + month + "/" + day, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(obj),
            credentials: "include"
        });
    }

    updateNote() {
        let notes = Data.notesDOM.value;
        this.notes[this.currentMonth+1] = notes;
        let month = "" + (this.currentMonth + 1);
        let year = "" + this.currentYear;
        let obj = {
            "data": {
                "note": notes
            }
        }
        fetch("/api/v1/note/" + year + "/" + month, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(obj),
            credentials: "include"
        });
    }

    updateDate(sameYear = false, refresh = true) {
        const url = window.location.href;
        const urlParts = url.split("/");
        const yearMonth = urlParts[urlParts.length - 1];
        const year = parseInt(yearMonth.substring(0, 4));
        const month = parseInt(yearMonth.substring(5, 7)) - 1;
        if (!isNaN(year) && !isNaN(month)) {
            this.currentMonth = month;
            this.currentYear = year;
        }
        if (!sameYear) {
            this.fetchData();
            this.fetchNotes();
        } else if (refresh) {
            this.refreshDOM();
        }
    }

    updateMonth(delta) {
        this.currentMonth += delta;
        if (this.currentMonth < 0) {
            this.currentMonth = 11;
            this.currentYear--;
        } else if (this.currentMonth > 11) {
            this.currentMonth = 0;
            this.currentYear++;
        }
        window.history.pushState({}, "", "/" + formatDate(this.currentYear, this.currentMonth));
        // Detect crossing a tracking-year boundary (0-indexed first/last tracking months).
        let firstMonth0 = (trackingStartMonth - 1) % 12;
        let lastMonth0 = (trackingStartMonth - 2 + 12) % 12;
        let sameYear = !(this.currentMonth === lastMonth0 && delta === -1) && !(this.currentMonth === firstMonth0 && delta === 1);
        this.updateDate(sameYear, true);
    }

    updateState(date, state) {
        if (!(this.currentMonth+1 in this.state)) {
            this.state[this.currentMonth+1] = {};
        }
        this.state[this.currentMonth+1][date] = state;
        this.updateBackend(date);
        
        // If setting to untracked, fetch fresh data to get the fallthrough scheduled state
        if (state === 0) {
            setTimeout(() => {
                this.fetchData();
            }, 100); // Small delay to ensure backend update completes
        } else {
            this.drawTarget();
        }
    }

    updateTitle() { Data.titleDOM.textContent = monthNames[this.currentMonth] + " " + this.currentYear; }
}

let rawState = {{ .YearlyState }};
let rawNotes = {{ .YearlyNotes }};
let state = mapState(rawState);
let notes = mapNotes(rawNotes);

let data = new Data(state, notes);

function generateCalendar(month, year, currState, callback) {
    let calendar = document.createElement("div");
    let table = document.createElement('table');
    let thead = document.createElement('thead');
    let tbody = document.createElement('tbody');
    const weekdays = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];
    let row = document.createElement('tr');
    weekdays.forEach(day => {
        let th = document.createElement('th');
        th.textContent = day;
        row.appendChild(th);
    });
    thead.appendChild(row);
    table.appendChild(thead);
    let firstDayOfMonth = new Date(year, month, 1);
    let startingDayOfWeek = firstDayOfMonth.getDay() || 7; // Convert Sunday from 0 to 7
    let currentDate = new Date(year, month, 1 - (startingDayOfWeek - 1));
    let today = new Date();
    today = new Date(today.getFullYear(), today.getMonth(), today.getDate());
    let first = true;
    while (first || currentDate.getMonth() === month || currentDate.getDay() !== 1) {
        first = false;
        row = document.createElement('tr');
        for (let i = 0; i < 7; i++) {
            let td = document.createElement('td');
            if (currentDate.getMonth() === month) {
                td.textContent = currentDate.getDate();
                td.classList.add('day');
                let cellState = 0;
                if (month+1 in currState && currentDate.getDate() in currState[month+1]) {
                    cellState = currState[month+1][currentDate.getDate()];
                }
                td.dataset.state = cellState; // Initial state
                td.classList.add(getClassForState(states[td.dataset.state]));
                
                if (currentDate.getTime() === today.getTime()) { td.classList.add('today'); }
                td.addEventListener('click', function () { callback(this, 1); });
                td.addEventListener('contextmenu', function (event) {
                    event.preventDefault();
                    callback(this, -1);
                });

                // Add tooltip events for running total
                let dayNum = currentDate.getDate();
                td.addEventListener('mouseenter', function(event) {
                    showTooltip(event, currState, month, year, dayNum);
                });
                td.addEventListener('mouseleave', hideTooltip);
            }
            row.appendChild(td);
            currentDate.setDate(currentDate.getDate() + 1);
        }
        tbody.appendChild(row);
    }
    table.appendChild(tbody);
    calendar.appendChild(table);
    return calendar;
}

function getClassForState(state) {
    switch (state) {
        case "untracked":
            return "untracked";
        case "present":
            return "present";
        case "not present":
            return "not-present";
        case "other":
            return "other";
        case "scheduled-present":
            return "scheduled-home";
        case "scheduled-not-present":
            return "scheduled-office";
        case "scheduled-other":
            return "scheduled-other";
        default:
            return "untracked";
    }
}

function mapState(payload) {
    let state = {};
    const months = payload.data.months;
    for (const [key, value] of Object.entries(months)) {
        if (!(key in state)) {
            state[key] = {};
        }

        let days = value.days;
        for (const [day, dayVal] of Object.entries(days)) {
            state[key][day] = dayVal.state;
        }
    }
    return state;
}

function mapNotes(payload) {
    let notes = {};
    for (const [key, value] of Object.entries(payload.data)) {
        notes[key] = value.note;
    }
    return notes;
}

function formatDate(year, month) { return year + "-" + (month + 1).toString().padStart(2, "0"); }

function calculateRunningTotal(currState, month, year, upToDay) {
    let presentDays = 0;
    let totalWorkDays = 0;

    // Calculate for all days in the month up to and including the specified day
    for (let day = 1; day <= upToDay; day++) {
        if (month + 1 in currState && day in currState[month + 1]) {
            let state = currState[month + 1][day];
            // States 2 and 5 are office days (actual and scheduled)
            if (state === 2 || state === 5) {
                presentDays++;
                totalWorkDays++;
            }
            // States 1 and 4 are WFH days (actual and scheduled)
            else if (state === 1 || state === 4) {
                totalWorkDays++;
            }
        }
    }

    let percentage = totalWorkDays > 0 ? ((presentDays / totalWorkDays) * 100).toFixed(1) : "0.0";
    return { presentDays, totalWorkDays, percentage };
}

function calculateAllTimeTotal(currState, currentMonth, upToDay) {
    let presentDays = 0;
    let totalWorkDays = 0;

    // Get all months from the state and sort them
    let months = Object.keys(currState).map(m => parseInt(m)).sort((a, b) => {
        // Order months within the tracking year (start month first).
        return trackingMonthOrder(a) - trackingMonthOrder(b);
    });

    for (let m of months) {
        let isCurrentMonth = (m === currentMonth + 1);
        let days = currState[m];

        for (let day in days) {
            let dayNum = parseInt(day);
            // For current month, only count up to the hovered day
            if (isCurrentMonth && dayNum > upToDay) {
                continue;
            }

            let state = days[day];
            // States 2 and 5 are office days (actual and scheduled)
            if (state === 2 || state === 5) {
                presentDays++;
                totalWorkDays++;
            }
            // States 1 and 4 are WFH days (actual and scheduled)
            else if (state === 1 || state === 4) {
                totalWorkDays++;
            }
        }

        // Stop after processing current month
        if (isCurrentMonth) {
            break;
        }
    }

    let percentage = totalWorkDays > 0 ? ((presentDays / totalWorkDays) * 100).toFixed(1) : "0.0";
    return { presentDays, totalWorkDays, percentage };
}

function showTooltip(event, currState, month, year, day) {
    // Remove any existing tooltip
    hideTooltip();

    let monthTotal = calculateRunningTotal(currState, month, year, day);
    let allTimeTotal = calculateAllTimeTotal(currState, month, day);

    let tooltip = document.createElement('div');
    tooltip.className = 'day-tooltip';
    tooltip.id = 'calendar-tooltip';
    tooltip.innerHTML = `<strong>Through ${monthNames[month]} ${day}:</strong><br>` +
        `Month: ${monthTotal.presentDays}/${monthTotal.totalWorkDays} days (${monthTotal.percentage}%)<br>` +
        `Year: ${allTimeTotal.presentDays}/${allTimeTotal.totalWorkDays} days (${allTimeTotal.percentage}%)`;

    document.body.appendChild(tooltip);

    // Position the tooltip above the hovered element
    let rect = event.target.getBoundingClientRect();
    let tooltipRect = tooltip.getBoundingClientRect();

    tooltip.style.left = (rect.left + rect.width / 2 - tooltipRect.width / 2 + window.scrollX) + 'px';
    tooltip.style.top = (rect.top - tooltipRect.height - 10 + window.scrollY) + 'px';
}

function hideTooltip() {
    let existing = document.getElementById('calendar-tooltip');
    if (existing) {
        existing.remove();
    }
}
