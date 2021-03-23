class App {
    #apiUrl = "";
    code = 0;

    setApiUrl(apiUrl) {
        this.#apiUrl = apiUrl
    }

    ping() {
        let worker = new Worker("assets/worker-ping.js");
        worker.onmessage = this.pingReceive
        worker.postMessage(this.#apiUrl + "ping")
        return this.code;
    }

    pingReceive(event) {
        this.code = event.data;
    }
}
