const assocUriOutput = document.getElementById("assoc-uri");
const assocUriInput = document.getElementById("generate-assoc-uri");
const assocUriCopy = document.getElementById("copy-assoc-uri");

let assocUri = "";

const generateAssocUri = _ => {
    fetch("/auth/generate-gh")
        .then(r => r.json())
        .then(payload => {
            assocUri = payload.secret;
            assocUriOutput.value = assocUri;
            assocUriCopy.innerText = "Copy";
        });
}

assocUriInput.addEventListener("click", generateAssocUri);

assocUriCopy.addEventListener("click", () => {
    navigator.clipboard.writeText(assocUriOutput.value).then(() => {
        assocUriCopy.innerText = "Copied";
    }).catch(function(error) {
        console.error('Failed to copy text: ', error);
    });
});