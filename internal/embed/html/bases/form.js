const states = ["untracked", "present", "not present", "other"];
const monthNames = ["January", "February", "March", "April", "May", "June", "July", "August", "September",
    "October", "November", "December"];

class Summary {
    constructor(data) { this.data = data || {}; }

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

        console.log(allTime);

        let headline = Data.summaryHeadlineDOM;
        let present = allTime[2];
        let total = allTime[1] + allTime[2];
        let percent = ((present / total * 100) || 0).toFixed(2);
        headline.textContent = `Present in office for ${present} out of ${total} days. (${percent}%)`;
    }

    updateMonth(year, month, state) {
        let key = formatDate(year, month);
        let stats = { 0: 0, 1: 0, 2: 0, 3: 0 };
        state.forEach(s => { stats[s] += 1; });
        this.data[key] = stats;
        this.refreshDOM();
    }
}

class Data {
    static titleDOM = document.getElementById("month-year");
    static calendarDOM = document.getElementById("calendar");
    static notesDOM = document.getElementById("notes");
    static summaryDOM = document.getElementById("summary-table");
    static summaryHeadlineDOM = document.getElementById("summary-headline");

    constructor(state, notes, summary) {
        this.state = state;
        this.notes = notes;
        this.summary = new Summary(summary);
        this.updateDate(false);
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

    drawNotes() { Data.notesDOM.value = this.notes; }

    drawSummary() { this.summary.refreshDOM(); }

    fetchData() {
        fetch("/api/v1/state/" + this.currentYear)
            .then(r => r.json())
            .then(payload => {
                console.log(payload);
                this.refreshDOM();
            });
        //
        // fetch("../user-state/" + formatDate(this.currentYear, this.currentMonth))
        //     .then(r => r.json())
        //     .then(payload => {
        //         this.state = payload.state;
        //         this.notes = payload.notes;
        //         this.refreshDOM();
        //     });
    }

    fetchNote() {
        fetch("/api/v1/note/" + this.currentYear + "/" + (this.currentMonth + 1))
            .then(r => r.json())
            .then(payload => {
                this.notes = payload.data.note;
                this.drawNotes();
            });
    }

    refreshDOM() {
        this.drawCalendar();
        this.drawNotes();
        this.drawSummary();
        this.updateTitle();
    }

    updateBackend() {
        let month = "" + (this.currentMonth + 1);
        let year = "" + this.currentYear;
        let days = {};
        this.state.forEach((s, i) => { i = "" + i; days[i] = s; });
        let notes = Data.notesDOM.value;
        let obj = {month, year, days, notes};
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
        let month = "" + (this.currentMonth + 1);
        let year = "" + this.currentYear;
        let notes = Data.notesDOM.value;
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

    updateDate(sameYear = false) {
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
        }
        this.fetchNote();
    }

    updateMonth(delta) {
        this.currentMonth += delta;
        let sameYear = false;
        if (this.currentMonth < 0) {
            this.currentMonth = 11;
            this.currentYear--;
        } else if (this.currentMonth > 11) {
            this.currentMonth = 0;
            this.currentYear++;
        } else {
            sameYear = true;
        }
        window.history.pushState({}, "", "/" + formatDate(this.currentYear, this.currentMonth));
        this.updateDate(sameYear);
    }

    updateState(date, state) {
        this.state[date] = state;
        this.updateBackend();
        this.summary.updateMonth(this.currentYear, this.currentMonth, this.state);
    }

    updateTitle() { Data.titleDOM.textContent = monthNames[this.currentMonth] + " " + this.currentYear; }
}

let state = []; // TODO: handle new state
let notes = "{{ .MonthNote.Note }}" // TODO: handle notes;
let summary = {}; // TODO: handle new summary

let yearlyState = {};

let data = new Data(state, notes, summary);

function generateCalendar(month, year, currState, callback) {
    console.log("Attempting to generate calendar for " + month + " " + year);
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
                td.dataset.state = currState[currentDate.getDate()]; // Initial state
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

function formatDate(year, month) { return year + "-" + (month + 1).toString().padStart(2, "0"); }