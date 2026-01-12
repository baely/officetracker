// ============ Service Worker for Officetracker Extension ============

console.log("Officetracker service worker started");

// ============ Alarm Setup ============

async function createAlarm() {
    const alarm = await chrome.alarms.get("autodetect-alarm");

    if (alarm) {
        console.log("Autodetect alarm already exists");
        return;
    }

    await chrome.alarms.create("autodetect-alarm", {
        periodInMinutes: 1
    });

    console.log("Autodetect alarm created (fires every 1 minute)");
}

// Create alarm on service worker startup
createAlarm();

// ============ Alarm Handler ============

chrome.alarms.onAlarm.addListener((alarm) => {
    if (alarm.name !== "autodetect-alarm") {
        return;
    }

    console.log("Autodetect alarm triggered at:", new Date().toISOString());

    // TODO: Future auto-detection implementation
    // This alarm will be used to automatically detect office presence
    // based on network conditions, location, or other signals.
    //
    // === PLANNED IMPLEMENTATION STEPS ===
    //
    // 1. CHECK USER AUTHENTICATION
    //    - Retrieve token from chrome.storage.local
    //    - If no token exists, skip auto-detection
    //    - Validate token is still active
    //
    // 2. DETECT CURRENT CONTEXT
    //    Implement one or more detection methods:
    //
    //    a) Network-based Detection:
    //       - Check connected WiFi SSID (requires additional permissions)
    //       - Compare against user's configured office WiFi patterns
    //       - Check for VPN connection status
    //       - Ping internal network endpoints (if accessible)
    //
    //    b) Location-based Detection:
    //       - Request geolocation coordinates (requires user permission)
    //       - Compare against office location (lat/long + radius)
    //       - Use IP-based geolocation as fallback
    //
    //    c) Time-based Detection:
    //       - Consider user's typical work hours
    //       - Different detection logic for workdays vs weekends
    //
    // 3. FETCH CURRENT STATUS
    //    - GET https://officetracker.com.au/api/v1/state/{year}/{month}/{day}
    //    - Headers: { "Authorization": "Bearer <token>" }
    //    - Parse response: data.data.state (0=Untracked, 1=WFH, 2=WFO, 3=Other)
    //
    // 4. DETERMINE ACTION
    //    - Compare detected context with current status
    //    - If context matches current status → no action needed
    //    - If context differs → proceed to step 5
    //
    // 5. UPDATE STATUS OR PROMPT USER
    //    Two possible approaches:
    //
    //    a) Auto-update (if user preference enabled):
    //       - PUT https://officetracker.com.au/api/v1/state/{year}/{month}/{day}
    //       - Headers: { "Authorization": "Bearer <token>", "Content-Type": "application/json" }
    //       - Body: { "data": { "state": <new-state> } }
    //       - Show notification confirming auto-update
    //
    //    b) Prompt for confirmation (default):
    //       - Create notification with action buttons
    //       - "Yes, update status" / "No, keep current"
    //       - Handle notification click events
    //       - Update status if user confirms
    //
    // 6. ERROR HANDLING
    //    - Network errors: Retry with exponential backoff
    //    - 401 Unauthorized: Clear token, show notification to re-login
    //    - Detection failures: Log and skip (don't update status)
    //    - Rate limiting: Respect API rate limits
    //
    // 7. LOGGING & DEBUGGING
    //    - Log all detection events for troubleshooting
    //    - Include timestamp, detected context, decision made
    //    - Store recent detection history for user review
    //
    // === REQUIRED PERMISSIONS (add to manifest.json) ===
    //
    // For geolocation-based detection:
    //   "permissions": [..., "geolocation"]
    //
    // For network/tab access:
    //   "permissions": [..., "tabs"]
    //   OR "permissions": [..., "activeTab"]
    //
    // For notifications:
    //   "permissions": [..., "notifications"]
    //
    // For WiFi SSID detection (not available in Chrome extensions):
    //   NOTE: Chrome extensions cannot access WiFi SSID directly
    //   Alternative: Use network endpoints or geolocation instead
    //
    // === USER PREFERENCES STRUCTURE ===
    //
    // Store in chrome.storage.local as "autodetect_preferences":
    // {
    //   enabled: boolean,                     // Master toggle for auto-detection
    //   autoUpdate: boolean,                  // Auto-update without confirmation
    //   detectionMethod: "location"|"network"|"both",
    //   officeLocation: {
    //     latitude: number,
    //     longitude: number,
    //     radius: number                      // Meters from office to consider "at office"
    //   },
    //   officeNetworks: [
    //     "office-wifi-name",
    //     "office-wifi-guest"
    //   ],
    //   workHours: {
    //     enabled: boolean,
    //     start: "09:00",
    //     end: "17:00",
    //     workdays: [1, 2, 3, 4, 5]           // 0=Sun, 1=Mon, ... 6=Sat
    //   },
    //   notificationEnabled: boolean,          // Show notifications on status change
    //   detectionSensitivity: "low"|"medium"|"high"
    // }
    //
    // === SETTINGS UI ===
    //
    // Create a settings page (options.html) for users to configure:
    // - Enable/disable auto-detection
    // - Set office location (with map picker)
    // - Set office WiFi networks
    // - Configure work hours
    // - Choose detection method
    // - Enable auto-update vs. prompt
    // - View detection history
    //
    // === EXAMPLE IMPLEMENTATION SNIPPET ===
    //
    // async function handleAutoDetection() {
    //     // 1. Get token and preferences
    //     const { officetracker_token, autodetect_preferences } =
    //         await chrome.storage.local.get(['officetracker_token', 'autodetect_preferences']);
    //
    //     if (!officetracker_token || !autodetect_preferences?.enabled) {
    //         return;
    //     }
    //
    //     // 2. Detect location
    //     const position = await new Promise((resolve, reject) => {
    //         navigator.geolocation.getCurrentPosition(resolve, reject);
    //     });
    //
    //     // 3. Calculate distance to office
    //     const distance = calculateDistance(
    //         position.coords.latitude,
    //         position.coords.longitude,
    //         autodetect_preferences.officeLocation.latitude,
    //         autodetect_preferences.officeLocation.longitude
    //     );
    //
    //     // 4. Determine status
    //     const isAtOffice = distance <= autodetect_preferences.officeLocation.radius;
    //     const detectedState = isAtOffice ? 2 : 1; // 2=WFO, 1=WFH
    //
    //     // 5. Get current status
    //     const { year, month, day } = getCurrentDate();
    //     const response = await fetch(
    //         `https://officetracker.com.au/api/v1/state/${year}/${month}/${day}`,
    //         { headers: { "Authorization": `Bearer ${officetracker_token}` } }
    //     );
    //     const { data } = await response.json();
    //     const currentState = data.state;
    //
    //     // 6. Update if different
    //     if (detectedState !== currentState) {
    //         if (autodetect_preferences.autoUpdate) {
    //             await updateStatus(detectedState);
    //             showNotification("Status updated automatically");
    //         } else {
    //             showNotificationWithActions(detectedState);
    //         }
    //     }
    // }
    //
    // // Call from alarm handler
    // handleAutoDetection().catch(console.error);
});

// ============ Extension Lifecycle ============

chrome.runtime.onInstalled.addListener(async ({ reason }) => {
    console.log("Extension installed/updated:", reason);

    if (reason === 'install') {
        console.log("First time installation - welcome to Officetracker!");
        // Could open onboarding page here
        // chrome.tabs.create({ url: chrome.runtime.getURL("welcome.html") });
    } else if (reason === 'update') {
        console.log("Extension updated to new version");
    }
});
