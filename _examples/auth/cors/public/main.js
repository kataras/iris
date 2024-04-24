// https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API/Using_Fetch
async function doRequest(method = 'GET', url = '', data = {}) {
    // Default options are marked with *

    const request = {
        method: method, // *GET, POST, PUT, DELETE, etc.
        mode: 'cors', // no-cors, *cors, same-origin
        cache: 'no-cache', // *default, no-cache, reload, force-cache, only-if-cached
        credentials: 'same-origin', // include, *same-origin, omit
        redirect: 'follow', // manual, *follow, error
        referrerPolicy: 'no-referrer', // no-referrer, *no-referrer-when-downgrade, origin, origin-when-cross-origin, same-origin, strict-origin, strict-origin-when-cross-origin, unsafe-url
    };

    if (data !== undefined && method !== 'GET' && method !== 'HEAD') {
        request.headers = {
            'Content-Type': 'application/json'
            // 'Content-Type': 'application/x-www-form-urlencoded',
        };
        // body data type must match "Content-Type" header.
        request.body = JSON.stringify(data);
    }

    const response = await fetch(url, request);
    return response.json(); // parses JSON response into native JavaScript objects.
}

const ul = document.getElementById("list");

function fetchData() {
    console.log("sending request...")

    doRequest('GET', '/data').then(data => {
        data.forEach(item => {
            var li = document.createElement("li");
            li.appendChild(document.createTextNode(item.title));
            ul.appendChild(li);
        });

        console.log(data); // JSON data parsed by `response.json()` call.
    });
}

document.getElementById("fetchBtn").onclick = fetchData;