class App {
    #apiUrl = "";

    setApiUrl(apiUrl) {
        this.#apiUrl = apiUrl
    }

    ping() {
        let req = new XMLHttpRequest();
        req.onload = function () {
            console.log(this.responseText);
        }
        req.onerror = function () {
            console.log(this);
        }
        req.open("GET", this.#apiUrl + "ping");
        req.send();
    }
}
