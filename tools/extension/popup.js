// ============ Constants ============

const API_BASE_URL = "https://officetracker.com.au";

// State mapping (colors match webapp at internal/embed/static/themes.css)
const STATE_MAP = {
    0: { name: "Untracked", display: "Not Set", class: "state-untracked", color: "#999" },
    1: { name: "WorkFromHome", display: "Work From Home", class: "state-wfh", color: "#4CAF50" },
    2: { name: "WorkFromOffice", display: "Work From Office", class: "state-wfo", color: "#F44336" },
    3: { name: "Other", display: "Other", class: "state-other", color: "#2196F3" }
};

// ============ Global Variables ============

const contents = document.getElementById("contents");

// ============ Authentication Functions ============

async function checkAuthentication() {
    console.log("[Officetracker] Checking authentication via cookie...");
    try {
        const response = await fetch(`${API_BASE_URL}/api/v1/settings/`, {
            method: "GET",
            credentials: "include"  // CRITICAL: Send cookies
        });

        console.log(`[Officetracker] Auth check response: ${response.status}`);
        return response.ok;  // 200 = authenticated, 401 = not authenticated
    } catch (error) {
        console.error("[Officetracker] Error checking authentication:", error);
        return false;
    }
}

// ============ API Functions ============

function getCurrentDate() {
    const now = new Date();
    return {
        year: now.getFullYear(),
        month: now.getMonth() + 1, // JS months are 0-indexed
        day: now.getDate()
    };
}

async function fetchTodayStatus() {
    const { year, month, day } = getCurrentDate();
    const url = `${API_BASE_URL}/api/v1/state/${year}/${month}/${day}`;
    console.log(`[Officetracker] Fetching status from: ${url}`);

    try {
        const response = await fetch(url, {
            method: "GET",
            credentials: "include"  // Send cookies automatically
        });

        console.log(`[Officetracker] Fetch response status: ${response.status} ${response.statusText}`);

        if (!response.ok) {
            if (response.status === 401) {
                console.error("[Officetracker] Unauthorized - cookie is invalid or expired");
                const error = new Error("Unauthorized");
                error.status = 401;
                throw error;
            }
            console.error(`[Officetracker] HTTP error: ${response.status}`);
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const data = await response.json();
        console.log("[Officetracker] Fetch response data:", data);
        console.log(`[Officetracker] Current state: ${data.data.state} (${STATE_MAP[data.data.state].display})`);
        return data.data.state; // Returns 0, 1, 2, or 3
    } catch (error) {
        console.error("[Officetracker] Error fetching status:", error);
        throw error;
    }
}

async function updateTodayStatus(state) {
    const { year, month, day } = getCurrentDate();
    const url = `${API_BASE_URL}/api/v1/state/${year}/${month}/${day}`;
    const payload = { data: { state: state } };

    console.log(`[Officetracker] Updating status to: ${state} (${STATE_MAP[state].display})`);
    console.log(`[Officetracker] PUT ${url}`);
    console.log("[Officetracker] Payload:", payload);

    try {
        const response = await fetch(url, {
            method: "PUT",
            headers: {
                "Content-Type": "application/json"
            },
            credentials: "include",  // Send cookies automatically
            body: JSON.stringify(payload)
        });

        console.log(`[Officetracker] Update response status: ${response.status} ${response.statusText}`);

        if (!response.ok) {
            if (response.status === 401) {
                console.error("[Officetracker] Unauthorized - cookie is invalid or expired");
                const error = new Error("Unauthorized");
                error.status = 401;
                throw error;
            }
            console.error(`[Officetracker] HTTP error: ${response.status}`);
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        console.log("[Officetracker] Status updated successfully");
        return true;
    } catch (error) {
        console.error("[Officetracker] Error updating status:", error);
        throw error;
    }
}

// ============ UI Helper Functions ============

function showError(message, containerId = "error-message") {
    const errorDiv = document.getElementById(containerId);
    if (errorDiv) {
        errorDiv.textContent = message;
        errorDiv.style.display = "block";
    }
}

function getErrorMessage(error) {
    if (error.status === 401) {
        return "Your session has expired. Please login again.";
    }
    if (error.status && error.status >= 500) {
        return "Server error. Please try again later.";
    }
    if (error.message && error.message.includes("Failed to fetch")) {
        return "Network error. Please check your internet connection.";
    }
    return "Failed to load data. Please refresh and try again.";
}

// ============ Event Handlers ============

async function handleStatusUpdate(newState) {
    console.log(`[Officetracker] === Status Update Started: ${newState} (${STATE_MAP[newState].display}) ===`);
    try {
        // Disable all buttons during update
        const buttons = document.querySelectorAll(".status-button");
        buttons.forEach(btn => btn.disabled = true);
        console.log("[Officetracker] Buttons disabled");

        await updateTodayStatus(newState);

        // Refresh UI to show new status
        console.log("[Officetracker] Refreshing UI...");
        await showLoggedInUI();
    } catch (error) {
        console.error("[Officetracker] Status update failed:", error);
        const message = getErrorMessage(error);
        showError(message, "status-error");

        // If unauthorized, show not logged in UI
        if (error.status === 401) {
            console.log("[Officetracker] Unauthorized - showing not logged in UI");
            setTimeout(() => {
                showNotLoggedInUI();
            }, 2000);
        } else {
            // Re-enable buttons on other errors
            console.log("[Officetracker] Re-enabling buttons after error");
            const buttons = document.querySelectorAll(".status-button");
            buttons.forEach(btn => btn.disabled = false);
        }
    }
}

// ============ UI Rendering Functions ============

function showNotLoggedInUI() {
    console.log("[Officetracker] Showing not-logged-in UI");
    contents.innerHTML = `
        <div class="not-logged-in">
            <h2>Not Logged In</h2>
            <p>You need to be logged into Officetracker to use this extension.</p>
            <p>
                <a href="https://officetracker.com.au" target="_blank" class="btn-primary">
                    Login to Officetracker
                </a>
            </p>
            <p class="help-text">After logging in, close and reopen this popup.</p>
        </div>
        <style>
            .not-logged-in { padding: 20px 0; text-align: center; }
            .not-logged-in h2 { font-size: 24px; margin-bottom: 16px; }
            .not-logged-in p { margin: 12px 0; line-height: 1.6; }
            .btn-primary { display: inline-block; background-color: #4CAF50; color: white; padding: 12px 24px; border-radius: 4px; text-decoration: none; font-weight: 500; }
            .btn-primary:hover { background-color: #45a049; }
            .help-text { font-size: 14px; color: #666; }
        </style>
    `;
}

function createLoggedInHTML(currentState) {
    const stateInfo = STATE_MAP[currentState];
    const today = new Date().toLocaleDateString('en-AU', {
        weekday: 'long',
        year: 'numeric',
        month: 'long',
        day: 'numeric'
    });

    return `
        <div class="status-container">
            <h2>Today's Status</h2>
            <p class="date">${today}</p>
            <div class="current-status ${stateInfo.class}">
                <span class="status-label">Current Status:</span>
                <span class="status-value">${stateInfo.display}</span>
            </div>
            <div class="actions">
                <h3>Quick Actions</h3>
                <div class="button-grid">
                    <button class="status-button btn-wfh" data-state="1">🏠 Work From Home</button>
                    <button class="status-button btn-wfo" data-state="2">🏢 Work From Office</button>
                    <button class="status-button btn-other" data-state="3">📍 Other</button>
                    <button class="status-button btn-untrack" data-state="0">❌ Untrack</button>
                </div>
            </div>
            <div id="status-error" class="error" style="display: none;"></div>
            <div class="footer">
                <a href="https://officetracker.com.au" target="_blank" class="link-app">Open Web App</a>
            </div>
        </div>
        <style>
            .status-container { padding: 20px 0; }
            .status-container h2 { margin-top: 0; font-size: 24px; margin-bottom: 8px; }
            .status-container h3 { font-size: 18px; margin-bottom: 12px; margin-top: 20px; }
            .date { color: #666; margin-bottom: 16px; font-size: 14px; }
            .current-status { padding: 20px; border-radius: 8px; margin: 16px 0; text-align: center; }
            .current-status .status-label { display: block; font-size: 12px; margin-bottom: 8px; text-transform: uppercase; letter-spacing: 0.5px; opacity: 0.8; }
            .current-status .status-value { display: block; font-size: 22px; font-weight: bold; }
            .state-untracked { background-color: #f5f5f5; color: #666; }
            .state-wfh { background-color: ${STATE_MAP[1].color}; color: white; }
            .state-wfo { background-color: ${STATE_MAP[2].color}; color: white; }
            .state-other { background-color: ${STATE_MAP[3].color}; color: white; }
            .actions { margin: 20px 0 16px 0; }
            .button-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }
            .status-button { padding: 12px; border: none; border-radius: 4px; font-size: 15px; cursor: pointer; transition: opacity 0.2s; font-weight: 500; }
            .status-button:hover:not(:disabled) { opacity: 0.85; }
            .status-button:disabled { opacity: 0.5; cursor: not-allowed; }
            .btn-wfh { background-color: ${STATE_MAP[1].color}; color: white; }
            .btn-wfo { background-color: ${STATE_MAP[2].color}; color: white; }
            .btn-other { background-color: ${STATE_MAP[3].color}; color: white; }
            .btn-untrack { background-color: #757575; color: white; }
            .footer { margin-top: 20px; padding-top: 16px; border-top: 1px solid #eee; text-align: center; }
            .link-app { color: #1976d2; text-decoration: none; font-size: 13px; }
            .link-app:hover { text-decoration: underline; }
            .error { color: #d32f2f; margin-top: 10px; padding: 10px; background-color: #ffebee; border-radius: 4px; font-size: 14px; }
        </style>
    `;
}

async function showLoggedInUI() {
    console.log("[Officetracker] === Showing Logged-In UI ===");
    // Show loading state
    contents.innerHTML = `<div style="padding: 20px; text-align: center;">
        <p>Loading...</p>
    </div>`;

    try {
        // Fetch current status
        const currentState = await fetchTodayStatus();

        // Create and display logged-in UI
        console.log("[Officetracker] Rendering logged-in UI...");
        contents.innerHTML = createLoggedInHTML(currentState);

        // Register event listeners
        const statusButtons = document.querySelectorAll(".status-button");
        console.log(`[Officetracker] Registering ${statusButtons.length} button listeners`);
        statusButtons.forEach(button => {
            button.addEventListener("click", () => {
                const newState = parseInt(button.dataset.state);
                handleStatusUpdate(newState);
            });
        });

        console.log("[Officetracker] Logged-in UI rendered successfully");
    } catch (error) {
        console.error("[Officetracker] Error showing logged-in UI:", error);
        const message = getErrorMessage(error);

        // If unauthorized, show not logged in UI
        if (error.status === 401) {
            console.log("[Officetracker] Cookie unauthorized - showing not logged in UI");
            showNotLoggedInUI();
        } else {
            console.log("[Officetracker] Showing error UI with retry option");
            contents.innerHTML = `<div class="error-container">
                <p>${message}</p>
                <button id="retry" class="btn-primary">Retry</button>
            </div>
            <style>
                .error-container { padding: 20px; text-align: center; }
                .error-container p { color: #d32f2f; margin-bottom: 20px; }
                .btn-primary { background-color: #4CAF50; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; font-size: 16px; width: 100%; }
                .btn-primary:hover { background-color: #45a049; }
            </style>`;

            document.getElementById("retry").addEventListener("click", showLoggedInUI);
        }
    }
}

// ============ Initialization ============

async function init() {
    console.log("[Officetracker] ========================================");
    console.log("[Officetracker] Extension initializing...");
    console.log("[Officetracker] API Base URL:", API_BASE_URL);

    try {
        // Check if user is authenticated via cookie
        const isAuthenticated = await checkAuthentication();

        if (isAuthenticated) {
            console.log("[Officetracker] User authenticated, showing status UI");
            await showLoggedInUI();
        } else {
            console.log("[Officetracker] User not authenticated");
            showNotLoggedInUI();
        }

        console.log("[Officetracker] Initialization complete");
        console.log("[Officetracker] ========================================");
    } catch (error) {
        console.error("[Officetracker] Initialization error:", error);
        contents.innerHTML = "<p style='padding: 20px; color: #d32f2f;'>Error loading extension. Please try again.</p>";
    }
}

// Start the extension
console.log("[Officetracker] Script loaded, starting initialization...");
init();
