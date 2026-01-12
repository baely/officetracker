// ============ Constants ============

const API_BASE_URL = "https://officetracker.com.au";
const STORAGE_KEY = "officetracker_token";

// State mapping (colors match webapp at internal/embed/static/themes.css)
const STATE_MAP = {
    0: { name: "Untracked", display: "Not Set", class: "state-untracked", color: "#999" },
    1: { name: "WorkFromHome", display: "Work From Home", class: "state-wfh", color: "#4CAF50" },
    2: { name: "WorkFromOffice", display: "Work From Office", class: "state-wfo", color: "#F44336" },
    3: { name: "Other", display: "Other", class: "state-other", color: "#2196F3" }
};

// ============ Global Variables ============

const contents = document.getElementById("contents");
let loginHTML;
let currentToken = null;

// ============ Logging Helper ============

function logToken(prefix, token) {
    // Only log first 20 chars of token for security
    const masked = token ? token.substring(0, 20) + "..." : "null";
    console.log(`[Officetracker] ${prefix}:`, masked);
}

// ============ Authentication Functions ============

async function isLoggedIn() {
    console.log("[Officetracker] Checking login status...");
    try {
        const result = await chrome.storage.local.get([STORAGE_KEY]);
        if (result[STORAGE_KEY]) {
            currentToken = result[STORAGE_KEY];
            logToken("Token found in storage", currentToken);
            return true;
        }
        console.log("[Officetracker] No token found in storage");
        return false;
    } catch (error) {
        console.error("[Officetracker] Error checking login status:", error);
        return false;
    }
}

function validateTokenFormat(token) {
    // Token should start with "officetracker:" followed by at least 60 characters
    // Allow alphanumeric plus common base64 characters (+, /, =)
    const tokenRegex = /^officetracker:[A-Za-z0-9+/=]{60,}$/;
    const isValid = tokenRegex.test(token);
    console.log(`[Officetracker] Token format validation: ${isValid ? "PASS" : "FAIL"}`);
    if (!isValid && token) {
        console.log(`[Officetracker] Token length: ${token.length}, starts with officetracker: ${token.startsWith('officetracker:')}`);
    }
    return isValid;
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
    logToken("Using token", currentToken);

    try {
        const response = await fetch(url, {
            method: "GET",
            headers: {
                "Authorization": `Bearer ${currentToken}`
            }
        });

        console.log(`[Officetracker] Fetch response status: ${response.status} ${response.statusText}`);

        if (!response.ok) {
            if (response.status === 401) {
                console.error("[Officetracker] Unauthorized - token is invalid or revoked");
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
    logToken("Using token", currentToken);

    try {
        const response = await fetch(url, {
            method: "PUT",
            headers: {
                "Authorization": `Bearer ${currentToken}`,
                "Content-Type": "application/json"
            },
            body: JSON.stringify(payload)
        });

        console.log(`[Officetracker] Update response status: ${response.status} ${response.statusText}`);

        if (!response.ok) {
            if (response.status === 401) {
                console.error("[Officetracker] Unauthorized - token is invalid or revoked");
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

function hideError(containerId = "error-message") {
    const errorDiv = document.getElementById(containerId);
    if (errorDiv) {
        errorDiv.style.display = "none";
    }
}

function getErrorMessage(error) {
    if (error.status === 401) {
        return "Your token has expired or been revoked. Please login again.";
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

async function handleLogin() {
    console.log("[Officetracker] === Login Flow Started ===");
    const tokenInput = document.getElementById("token");
    const loginButton = document.getElementById("login");
    const token = tokenInput.value.trim();

    logToken("Attempting login with token", token);
    hideError();

    // Validate non-empty
    if (!token) {
        console.log("[Officetracker] Login failed: empty token");
        showError("Please enter an API token.");
        return;
    }

    // Validate format
    if (!validateTokenFormat(token)) {
        console.log("[Officetracker] Login failed: invalid token format");
        showError("Invalid token format. Token should start with 'officetracker:' followed by at least 60 characters.");
        return;
    }

    // Disable button and save token
    loginButton.disabled = true;
    loginButton.textContent = "Logging in...";

    // Save token
    try {
        console.log("[Officetracker] Saving token to storage...");
        await chrome.storage.local.set({ [STORAGE_KEY]: token });
        currentToken = token;
        console.log("[Officetracker] Token saved successfully");

        // Try to show logged-in UI (this will validate the token with the API)
        console.log("[Officetracker] Attempting to load logged-in UI...");
        await showLoggedInUI();
    } catch (error) {
        console.error("[Officetracker] Login failed:", error);
        showError("Failed to save token. Please try again.");
        loginButton.disabled = false;
        loginButton.textContent = "Login";
    }
}

async function handleLogout() {
    console.log("[Officetracker] Logging out...");
    try {
        await chrome.storage.local.remove(STORAGE_KEY);
        currentToken = null;
        console.log("[Officetracker] Token removed from storage");
        showLoginUI();
    } catch (error) {
        console.error("[Officetracker] Error logging out:", error);
    }
}

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

        // If unauthorized, clear token and show login
        if (error.status === 401) {
            console.log("[Officetracker] Unauthorized - clearing token and returning to login");
            await chrome.storage.local.remove(STORAGE_KEY);
            currentToken = null;
            setTimeout(() => {
                showLoginUI();
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
                <button class="status-button btn-wfh" data-state="1">🏠 Work From Home</button>
                <button class="status-button btn-wfo" data-state="2">🏢 Work From Office</button>
                <button class="status-button btn-other" data-state="3">📍 Other</button>
                <button class="status-button btn-untrack" data-state="0">❌ Untrack</button>
            </div>
            <div id="status-error" class="error" style="display: none;"></div>
            <div class="footer">
                <button id="logout" class="btn-logout">Logout</button>
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
            .status-button { display: block; width: 100%; padding: 12px; margin: 8px 0; border: none; border-radius: 4px; font-size: 15px; cursor: pointer; transition: opacity 0.2s; font-weight: 500; }
            .status-button:hover:not(:disabled) { opacity: 0.85; }
            .status-button:disabled { opacity: 0.5; cursor: not-allowed; }
            .btn-wfh { background-color: ${STATE_MAP[1].color}; color: white; }
            .btn-wfo { background-color: ${STATE_MAP[2].color}; color: white; }
            .btn-other { background-color: ${STATE_MAP[3].color}; color: white; }
            .btn-untrack { background-color: #757575; color: white; }
            .footer { margin-top: 20px; padding-top: 16px; border-top: 1px solid #eee; display: flex; justify-content: space-between; align-items: center; }
            .btn-logout { background-color: #757575; color: white; padding: 8px 16px; border: none; border-radius: 4px; cursor: pointer; font-size: 13px; }
            .btn-logout:hover { background-color: #616161; }
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

        const logoutButton = document.getElementById("logout");
        logoutButton.addEventListener("click", handleLogout);

        console.log("[Officetracker] Logged-in UI rendered successfully");
    } catch (error) {
        console.error("[Officetracker] Error showing logged-in UI:", error);
        const message = getErrorMessage(error);

        // If unauthorized, clear token and show login
        if (error.status === 401) {
            console.log("[Officetracker] Token unauthorized - clearing and showing login");
            await chrome.storage.local.remove(STORAGE_KEY);
            currentToken = null;
            showLoginUI();
            setTimeout(() => {
                showError("Token is invalid or has been revoked. Please login again.");
            }, 100);
        } else {
            console.log("[Officetracker] Showing error UI with retry option");
            contents.innerHTML = `<div class="error-container">
                <p>${message}</p>
                <button id="retry" class="btn-primary">Retry</button>
                <button id="logout-error" class="btn-logout" style="margin-top: 10px;">Logout</button>
            </div>
            <style>
                .error-container { padding: 20px; text-align: center; }
                .error-container p { color: #d32f2f; margin-bottom: 20px; }
                .btn-primary { background-color: #4CAF50; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; font-size: 16px; width: 100%; }
                .btn-primary:hover { background-color: #45a049; }
                .btn-logout { background-color: #f44336; color: white; padding: 8px 16px; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; width: 100%; }
                .btn-logout:hover { background-color: #d32f2f; }
            </style>`;

            document.getElementById("retry").addEventListener("click", showLoggedInUI);
            document.getElementById("logout-error").addEventListener("click", handleLogout);
        }
    }
}

function showLoginUI() {
    console.log("[Officetracker] Showing login UI");
    contents.innerHTML = loginHTML;

    // Register event listeners
    const loginButton = document.getElementById("login");
    const tokenInput = document.getElementById("token");

    loginButton.addEventListener("click", handleLogin);
    tokenInput.addEventListener("keypress", (e) => {
        if (e.key === "Enter") {
            handleLogin();
        }
    });
    console.log("[Officetracker] Login UI rendered, event listeners attached");
}

// ============ Initialization ============

async function init() {
    console.log("[Officetracker] ========================================");
    console.log("[Officetracker] Extension initializing...");
    console.log("[Officetracker] API Base URL:", API_BASE_URL);

    try {
        // Load login template
        console.log("[Officetracker] Loading login.html template...");
        const response = await fetch(chrome.runtime.getURL("./login.html"));
        loginHTML = await response.text();
        console.log("[Officetracker] Login template loaded successfully");

        // Check if user is logged in
        const loggedIn = await isLoggedIn();

        if (loggedIn) {
            console.log("[Officetracker] User is logged in, showing logged-in UI");
            await showLoggedInUI();
        } else {
            console.log("[Officetracker] User not logged in, showing login UI");
            showLoginUI();
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
