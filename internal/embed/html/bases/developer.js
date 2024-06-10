const apiKeyOutput = document.getElementById("api-key");
const apiKeyInput = document.getElementById("generate-api-key");
const apiKeyCopy = document.getElementById("copy-api-key");

let apiKey = "";

const generateApiKey = _ => {
    fetch("/api/v1/developer/secret")
        .then(r => r.json())
        .then(payload => {
            apiKey = payload.secret;
            apiKeyOutput.value = apiKey;
            apiKeyCopy.innerText = "Copy";
        });
}

apiKeyInput.addEventListener("click", () => {
    let ok = confirm("Are you sure you want to generate a new API key and revoke all previous API keys?");
    if (!ok) {
        return;
    }
    generateApiKey();
});

apiKeyCopy.addEventListener("click", () => {
    navigator.clipboard.writeText(apiKeyOutput.value).then(() => {
        apiKeyCopy.innerText = "Copied";
    }).catch(function(error) {
        console.error('Failed to copy text: ', error);
    });
});