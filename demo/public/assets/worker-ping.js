onmessage = function(event) {
    let code = 0;
    let req = new XMLHttpRequest();
    req.onload = function () {
        let resp = JSON.parse(this.response);
        code = resp.status;
    }
    req.onerror = function () {
        code = 500;
    }
    req.open("GET", event.data, false);
    req.send();

    postMessage(code);
};
