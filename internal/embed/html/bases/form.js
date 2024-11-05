const states = ["untracked", "present", "not present", "other"];
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

        let allTime = { 0: 0, 1: 0, 2: 0, 3: 0 };

        if (this.data == null) { return; }
        let keys = Object.keys(this.data);
        keys.sort();
        for (let i = 0; i < keys.length; i++) {
            let key = keys[i];
            let stats = this.data[key];
            for (let j = 0; j < 4; j++) {
                allTime[j] += stats[j];
            }
            let monthYear = key.split("-");
            let month = parseInt(monthYear[1]) - 1;
            let year = parseInt(monthYear[0]);

            let row = document.createElement("tr");
            let monthCell = document.createElement("td");
            monthCell.textContent = monthNames[month] + " " + year;
            row.appendChild(monthCell);

            let presentCell = document.createElement("td");
            presentCell.textContent = stats[2];
            row.appendChild(presentCell);

            let totalCell = document.createElement("td");
            totalCell.textContent = stats[1] + stats[2];
            row.appendChild(totalCell);

            let percentCell = document.createElement("td");
            percentCell.textContent = (stats[2] / (stats[1] + stats[2]) * 100).toFixed(2) + "%";
            row.appendChild(percentCell);

            if (stats[1] + stats[2] <= 0) { continue; }

            elem.appendChild(row);
        }
        let headline = Data.summaryHeadlineDOM;
        let present = allTime[2];
        let total = allTime[1] + allTime[2];
        let percent = ((present / total * 100) || 0).toFixed(2);
        headline.textContent = `Present in office for ${present} out of ${total} days. (${percent}%)`;
    }

    updateMonth(year, month, state) {
        let key = formatDate(year, month);
        let stats = { 0: 0, 1: 0, 2: 0, 3: 0 };
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
            let stats = { 0: 0, 1: 0, 2: 0, 3: 0 };
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
        let previousState = parseInt(dayDOM.dataset.state);
        let currentState = (previousState + direction + states.length) % states.length;
        let date = dayDOM.textContent;
        dayDOM.classList.remove(getClassForState(states[previousState]));
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
        this.summary.updateMonth(this.currentYear, this.currentMonth, this.state);
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

function generatePdf() {

}