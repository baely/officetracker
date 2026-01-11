// DOM elements
const tokenNameInput = document.getElementById("token-name");
const generateTokenBtn = document.getElementById("generate-token-btn");
const newTokenDisplay = document.getElementById("new-token-display");
const newTokenValue = document.getElementById("new-token-value");
const copyTokenBtn = document.getElementById("copy-token-btn");
const tokensList = document.getElementById("tokens-list");

// Load tokens on page load
loadTokens();

// Generate new token
generateTokenBtn.addEventListener("click", async () => {
    const name = tokenNameInput.value.trim();
    if (!name) {
        alert("Please enter a name for your token");
        return;
    }

    try {
        const response = await fetch("/api/v1/developer/secret", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ data: { name } })
        });

        if (!response.ok) {
            throw new Error("Failed to generate token");
        }

        const data = await response.json();

        // Show the new token
        newTokenValue.value = data.secret;
        newTokenDisplay.style.display = "block";
        copyTokenBtn.innerText = "Copy";

        // Clear the input
        tokenNameInput.value = "";

        // Reload the tokens list
        loadTokens();
    } catch (error) {
        console.error("Error generating token:", error);
        alert("Failed to generate token. Please try again.");
    }
});

// Copy token to clipboard
copyTokenBtn.addEventListener("click", async () => {
    try {
        await navigator.clipboard.writeText(newTokenValue.value);
        copyTokenBtn.innerText = "Copied!";
        setTimeout(() => {
            copyTokenBtn.innerText = "Copy";
        }, 2000);
    } catch (error) {
        console.error("Failed to copy:", error);
        alert("Failed to copy to clipboard");
    }
});

// Load and display tokens
async function loadTokens() {
    try {
        const response = await fetch("/api/v1/developer/tokens");
        if (!response.ok) {
            throw new Error("Failed to load tokens");
        }

        const data = await response.json();
        renderTokens(data.tokens || []);
    } catch (error) {
        console.error("Error loading tokens:", error);
        tokensList.innerHTML = '<p class="error">Failed to load tokens</p>';
    }
}

// Render tokens list
function renderTokens(tokens) {
    if (tokens.length === 0) {
        tokensList.innerHTML = '<p class="no-tokens">No active tokens. Create one above to get started.</p>';
        return;
    }

    const tableRows = tokens.map(token => `
        <tr>
            <td>${escapeHtml(token.name)}</td>
            <td>${formatRelativeTime(token.created_at)}</td>
            <td style="text-align: right;">
                <button class="revoke-btn" data-token-id="${token.token_id}" data-token-name="${escapeHtml(token.name)}">
                    Revoke
                </button>
            </td>
        </tr>
    `).join('');

    tokensList.innerHTML = `
        <table class="tokens-table">
            <thead>
                <tr>
                    <th>Name</th>
                    <th>Created</th>
                    <th style="text-align: right;">Actions</th>
                </tr>
            </thead>
            <tbody>
                ${tableRows}
            </tbody>
        </table>
    `;

    // Add event listeners to revoke buttons
    document.querySelectorAll('.revoke-btn').forEach(btn => {
        btn.addEventListener('click', () => revokeToken(
            btn.dataset.tokenId,
            btn.dataset.tokenName
        ));
    });
}

// Revoke a token
async function revokeToken(tokenId, tokenName) {
    const confirmed = confirm(`Are you sure you want to revoke the token "${tokenName}"? This action cannot be undone.`);
    if (!confirmed) return;

    try {
        const response = await fetch(`/api/v1/developer/tokens/${tokenId}`, {
            method: "DELETE"
        });

        if (!response.ok) {
            throw new Error("Failed to revoke token");
        }

        // Reload the tokens list
        loadTokens();

        // Hide new token display if it was showing
        newTokenDisplay.style.display = "none";
    } catch (error) {
        console.error("Error revoking token:", error);
        alert("Failed to revoke token. Please try again.");
    }
}

// Format relative time
function formatRelativeTime(isoString) {
    const date = new Date(isoString);
    const now = new Date();
    const seconds = Math.floor((now - date) / 1000);

    if (seconds < 60) return 'just now';
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return `${minutes} minute${minutes === 1 ? '' : 's'} ago`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `${hours} hour${hours === 1 ? '' : 's'} ago`;
    const days = Math.floor(hours / 24);
    if (days < 30) return `${days} day${days === 1 ? '' : 's'} ago`;
    const months = Math.floor(days / 30);
    if (months < 12) return `${months} month${months === 1 ? '' : 's'} ago`;
    const years = Math.floor(months / 12);
    return `${years} year${years === 1 ? '' : 's'} ago`;
}

// Escape HTML to prevent XSS
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
