const HTTP_API_URL = "http://127.0.0.1:9045"

// fetch the data from the API server
// returns a promise for the json data.
async function fetchMsgHistory(reqBody){
    return fetch(HTTP_API_URL + "?reqType=MessageHistory", {
        method: "POST",
        body: JSON.stringify(reqBody)
    })
    .then((res) => res.json())
}

async function fetchConnectedDevices(){
    return fetch(HTTP_API_URL + "?reqType=GetConnectedDevices", {
        method: "GET",
    })
    .then((res) => res.json())
}

export {
    fetchMsgHistory,
    fetchConnectedDevices
}