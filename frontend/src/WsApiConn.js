

const WS_API_SVR_ENDPOINT = "ws://127.0.0.1:9046"

// singleton pattern for the persistent websocket 
class ApiSvrConnection {
    constructor(){
        // asssign the promise returned by the connect function to a member
        this.connectPromise = this.connect()
        // timeout in MS
        this.timeout = 1000;
    }

    // set the function which is called when data is received
    setReceiveCallback = (callback) => {
        this.processReceived = callback
    }
    
    // connects the websocket, returning a promise for the succesful connection.
    connect = () => {
        return new Promise((resolve, reject) => {
            this.apiConnection = new WebSocket(WS_API_SVR_ENDPOINT, ["dvr_api"])
            this.apiConnection.onopen = (event) => {
                console.log("WS connected to API server");
                resolve(this)
            }
            this.apiConnection.onmessage = (event) => {
                this.processReceived(event);
            }
            const handleDisconnect = (event) => {
                console.log("WS disconnected from API server");
                delete this.apiConnection; // gc
                setTimeout(this.connect, this.timeout += this.timeout) // exponentially increase timeout
                reject(this)
            }
            this.apiConnection.onclose = handleDisconnect
            this.apiConnection.onerror = handleDisconnect
        })
    }
    // singleton pattern to restrict instances
    static getInstance() {
        if (!ApiSvrConnection.instance) ApiSvrConnection.instance = new ApiSvrConnection();
        return ApiSvrConnection.instance;
    }
}
export default ApiSvrConnection.getInstance();

// this.apiSvrConnection = new WebSocket(URL);
// this.apiSvrConnection.onopen = (event) => {
//     console.log("Connected to API server: ", event.data)
// }
// this.apiSvrConnection.onerror = (event) => {
//     console.log("Error in websocket connection to API server: ", event.data)
// }
// this.apiSvrConnection.onclose = (event) => {
//     console.log("Disconnected from API server: ", event.data);
// }