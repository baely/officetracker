const states = ["untracked", "present", "not present", "other", "scheduled-present", "scheduled-not-present", "scheduled-other"];
const monthNames = ["January", "February", "March", "April", "May", "June", "July", "August", "September",
    "October", "November", "December"];

class Summary {
    constructor(data, year) {
        this.updateYear(year, data);
    }

    refreshDOM() {
        const elem = Data.summaryDOM;
        const headings = ["Month", "Present", "Total", "Percent"];
        elem.innerHTML = "";
        let header = document.createElement("tr");
        headings.forEach(heading => {
            let th = document.createElement("th");
            th.textContent = heading;
            header.appendChild(th);
        });
        elem.appendChild(header);

        let allTime = { 0: 0, 1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 6: 0 };

        if (this.data == null) { return; }
        let keys = Object.keys(this.data);
        keys.sort();
        for (let i = 0; i < keys.length; i++) {
            let key = keys[i];
            let stats = this.data[key];
            for (let j = 0; j < 7; j++) {
                allTime[j] += (stats[j] || 0);
            }
            let monthYear = key.split("-");
            let month = parseInt(monthYear[1]) - 1;
            let year = parseInt(monthYear[0]);

            let row = document.createElement("tr");
            let monthCell = document.createElement("td");
            monthCell.textContent = monthNames[month] + " " + year;
            row.appendChild(monthCell);

            let presentCell = document.createElement("td");
            // Count actual office days + scheduled office days
            let actualPresent = stats[2] || 0;
            let scheduledPresent = stats[5] || 0; // StateScheduledWorkFromOffice
            presentCell.textContent = actualPresent + scheduledPresent;
            row.appendChild(presentCell);

            let totalCell = document.createElement("td");
            // Count all work days (WFH + Office, both actual and scheduled)
            let totalExpected = (stats[1] || 0) + (stats[2] || 0) + (stats[4] || 0) + (stats[5] || 0);
            totalCell.textContent = totalExpected;
            row.appendChild(totalCell);

            let percentCell = document.createElement("td");
            let percentage = totalExpected > 0 ? ((actualPresent + scheduledPresent) / totalExpected * 100).toFixed(2) : "0.00";
            percentCell.textContent = percentage + "%";
            row.appendChild(percentCell);

            if (totalExpected <= 0) { continue; }

            elem.appendChild(row);
        }
        let headline = Data.summaryHeadlineDOM;
        let actualPresent = allTime[2] || 0;
        let scheduledPresent = allTime[5] || 0;
        let totalPresent = actualPresent + scheduledPresent;
        let total = (allTime[1] || 0) + (allTime[2] || 0) + (allTime[4] || 0) + (allTime[5] || 0);
        let percent = total > 0 ? ((totalPresent / total * 100) || 0).toFixed(2) : "0.00";
        headline.textContent = `Present in office for ${totalPresent} out of ${total} days. (${percent}%)`;
    }

    updateMonth(year, month, state) {
        let key = formatDate(year, month);
        let stats = { 0: 0, 1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 6: 0 };
        for (let day in state[month + 1]) {
            stats[state[month + 1][day]] += 1;
        }
        
        this.data[key] = stats;
        this.refreshDOM();
    }

    updateYear(year, state) {
        this.year = year;
        this.data = {};
        for (const [month, vals] of Object.entries(state)) {
            let monthYear = month <= 9 ? year : year - 1;
            let key = formatDate(monthYear, parseInt(month) - 1);
            let stats = { 0: 0, 1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 6: 0 };
            for (const [day, state] of Object.entries(vals)) {
                stats[state] += 1;
            }
            
            this.data[key] = stats;
        }
        this.refreshDOM();
    }
    
}

class Data {
    static titleDOM = document.getElementById("month-year");
    static calendarDOM = document.getElementById("calendar");
    static notesDOM = document.getElementById("notes");
    static summaryDOM = document.getElementById("summary-table");
    static summaryHeadlineDOM = document.getElementById("summary-headline");

    constructor(state, notes) {
        this.state = state;
        this.notes = notes;
        this.updateDate(true, false);
        let year = this.currentMonth < 9 ? this.currentYear : this.currentYear + 1;
        this.summary = new Summary(state, year);
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

    drawSummary() { this.summary.refreshDOM(); }

    fetchData() {
        let year = this.currentMonth < 9 ? this.currentYear : this.currentYear + 1;
        fetch("/api/v1/state/" + year)
            .then(r => r.json())
            .then(payload => {
                this.state = mapState(payload);
                this.refreshDOM();
                this.summary.updateYear(year, this.state);
            });
    }

    fetchNotes() {
        let year = this.currentMonth < 9 ? this.currentYear : this.currentYear + 1;
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
        this.drawSummary();
        this.updateTitle();
        this.updateReportButtons();
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
        let sameYear = !(this.currentMonth === 8 && delta === -1) && !(this.currentMonth === 9 && delta === 1);
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
            this.summary.updateMonth(this.currentYear, this.currentMonth, this.state);
        }
    }

    updateTitle() { Data.titleDOM.textContent = monthNames[this.currentMonth] + " " + this.currentYear; }

    updateReportButtons() {
        const csvButton = document.getElementById("export-csv");
        const pdfButton = document.getElementById("export-pdf");

        if (csvButton.onclick) { csvButton.onclick = null; }
        if (pdfButton.onclick) { pdfButton.onclick = null; }

        let year = this.currentMonth < 9 ? this.currentYear : this.currentYear + 1;

        csvButton.onclick = () => {
            window.location.href = "/api/v1/report/csv/" + year + "-attendance";
        }

        pdfButton.onclick = () => {
            let name = prompt("(Optional) Please enter your name", "");
            window.location.href = "/api/v1/report/pdf/" + year + "-attendance?name=" + name;
        }
    }
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
        // Fiscal year ordering: Oct(10), Nov(11), Dec(12), Jan(1), Feb(2), etc.
        let aOrder = a >= 10 ? a - 10 : a + 2;
        let bOrder = b >= 10 ? b - 10 : b + 2;
        return aOrder - bOrder;
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
