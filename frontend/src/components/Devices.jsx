﻿// external
import React, { useState, useEffect } from 'react';
import { Button, List, Input } from 'reactstrap';

// js
import WsApiConn from '../WsApiConn';
import { getCurrentTime, formatDateTimeDVRFormat } from '../Utils';

// components
import { TabContent, TabButtons } from './Tabs';
import VidReq from './VidReq';
import { MsgHistoryGrid } from './MsgHistoryGrid';

export default function Devices() {

    // states
    const [ msgVal, setMsgVal ] = useState("")
    const [ devList, setDevList ] = useState([]);
    const [ selectedDevice, setSelectedDevice ] = useState("");
    const [ activeTab, setActiveTab ] = useState(0);

    // tabs
    const tabData = [
        {
            title: "Custom",
            component: null
        },
        {
            title: "Video Request",
            component: <VidReq setMsgVal={setMsgVal}/>
        },
    ];


    function WsApiConnection_Callback(event) {
        // parse the message we've received over the WS connection
        const payload = JSON.parse(event.data)
        // decide what to do with the packet
        if ("ConnectedDevicesList" in payload) {
            setDevList(payload.ConnectedDevicesList)
            WsApiConn.apiConnection.send(JSON.stringify({
                "Subscriptions": payload.ConnectedDevicesList
            }))
        }
        else if ("Message" in payload) {
            addMessageToLog(payload.Message)
        }
    }

    // do things at start of the page load
    useEffect(() => {
        // set what we want to do with received data
        WsApiConn.setReceiveCallback(WsApiConnection_Callback)
        // when the connect promise resolves send a request for the connected device list
        WsApiConn.connectPromise.then(() => {
            WsApiConn.apiConnection.send(JSON.stringify({
                "GetConnectedDevices": true
            }))
        }).catch((err) => {
            console.log(err)
        })
    }, []);

    // 
    const addMessageToLog = (message) => {
        const li = document.createElement("li");
        li.appendChild(document.createTextNode(getCurrentTime() + ": " + message));
        const firstItem = document.getElementById("message-response-list").firstChild;
        if (firstItem) {
            document.getElementById("message-response-list").insertBefore(li, firstItem);
        } else {
            document.getElementById("message-response-list").appendChild(li);
        }
    }

    // event handler for when the selected device is changed
    const handleDevSelection = (event) => { 
        setSelectedDevice(event.target.options[event.target.selectedIndex].value) 
    };

    // send the message to the API server
    const sendToApiSvr = () => {
        const message = document.querySelector("#send-message-input").value + "\r";
        WsApiConn.apiConnection.send(JSON.stringify({
            "Messages": [message]
        }))
    }

    return (
        <div>
            
            {/* select type of message you want to send */}
            <div className="tabs_container">
                <h5>Message Template</h5>
                <TabButtons
                    activeTab={activeTab}
                    setActiveTab={setActiveTab}
                    tabData={tabData}
                />
                <TabContent 
                    activeTab={activeTab}
                    tabData={tabData}
                />
            </div>

            {/* configure the message */}
            <div>
                <h5>Configure Message</h5>
                <label htmlFor="device-selector">Device: </label>
                <Input id="device-selector" type="select" className="api-message-param" onChange={handleDevSelection} required>
                    {/* initial value of a select option isn't technically selected for some reason */}
                    {devList && devList.length > 0 ? (
                        devList.map((id) => (
                            <option key={id} value={id}>
                                {id}
                            </option>
                        ))
                    ) : (
                        <option disabled>No devices connected to server</option>
                    )}
                </Input>
                <br />
                <Input type="text" id="send-message-input" defaultValue={msgVal}/>
                <small>^reload the page to reset the input box.</small>
                <br />
                <Button id="send-message-button" onClick={sendToApiSvr} color="primary">Send</Button>
                <br />
                <List id="message-response-list"></List>
            </div>
            
            {/* display the message and the context it resides in */}
            <div>
                <h5>Message History</h5>
                <MsgHistoryGrid 
                    device={selectedDevice} 
                    after={"2023-10-03T16:45:14.000+00:00"} 
                    before={"2025-10-03T16:45:14.000+00:00"}
                />
            </div>
        </div>
    );
}


// // external
// import React, { useState, useEffect } from 'react';
// import { Button, List, Input } from 'reactstrap';

// // js
// import WsApiConn from '../WsApiConn';
// import { getCurrentTime, formatDateTimeDVRFormat } from '../Utils';

// // components
// import { TabContent, TabButtons } from './Tabs';
// import VidReq from './VidReq';
// import { MsgHistoryGrid } from './MsgHistoryGrid';

// export default function Devices() {

//     // states
//     const [ msgVal, setMsgVal ] = useState("") // val of message in message box
//     const [ devList, setDevList ] = useState(["Select a device"]); // devices in the selection box
//     const [ selectedDevice, setSelectedDevice ] = useState(""); // changes when we select a different device
//     const [ activeTab, setActiveTab ] = useState(0); // tabs, custom and vidreq

//     // tabs
//     const tabData = [
//         {
//             title: "Custom",
//             component: null
//         },
//         {
//             title: "Video Request",
//             component: <VidReq setMsgVal={setMsgVal}/>
//         },
//     ];

//     // do things at start of the page load
//     useEffect(() => {
//         // set what we want to do with received data
//         WsApiConn.setReceiveCallback(Devices_ApiConnectionCallback)
//         // when the connect promise resolves send a request for the connected device list
//         WsApiConn.connectPromise.then(() => {
//             WsApiConn.apiConnection.send(JSON.stringify({
//                 "GetConnectedDevices": true,
//                 "Subscriptions": [selectedDevice]
//             }))
//         }).catch((err) => {
//             console.log(err)
//         })
//     }, []);

//     // whenever we change the selected device then resubscribe to it
//     useEffect(() => {
//         WsApiConn.connectPromise.then(() => {
//             WsApiConn.apiConnection.send(JSON.stringify({
//                 "Subscriptions": [selectedDevice]
//             }))
//         })
//     }, [selectedDevice])

//     // this function fires on every event received over the API connection.
//     function Devices_ApiConnectionCallback(event) {
//         // parse the message we've received over the WS connection
//         const payload = JSON.parse(event.data)
//         // decide what to do with the packet
//         if ('ConnectedDevicesList' in payload) {
//             setDevList(payload.connectedDevicesList)
//         }
//         else if ("Message" in payload) {
//             addMessageToLog(payload.message)
//             console.log(payload.message)
//         }
//     }

//     // show the message we've subscribed to
//     const addMessageToLog = (message) => {
//         const li = document.createElement("li");
//         li.appendChild(document.createTextNode(getCurrentTime() + ": " + message));
//         const firstItem = document.getElementById("message-response-list").firstChild;
//         if (firstItem) {
//             document.getElementById("message-response-list").insertBefore(li, firstItem);
//         } else {
//             document.getElementById("message-response-list").appendChild(li);
//         }
//     }

//     // event handler for when the selected device is changed
//     const handleDevSelection = (event) => { 
//         setSelectedDevice(event.target.options[event.target.selectedIndex].value)
//         WsApiConn.apiConnection.send(JSON.stringify({
//             "Subscriptions": [selectedDevice]
//         }))
//     };

//     // send the message to the API server
//     const sendToApiSvr = () => {
//         const message = document.querySelector("#send-message-input").value + "\r";
//         WsApiConn.apiConnection.send(JSON.stringify({
//             "Messages": [message],
//             "Subscriptions": [selectedDevice]
//         }))
//     }

//     return (
//         <div>
            
//             {/* select type of message you want to send */}
//             <div className="tabs_container">
//                 <h5>Message Template</h5>
//                 <TabButtons
//                     activeTab={activeTab}
//                     setActiveTab={setActiveTab}
//                     tabData={tabData}
//                 />
//                 <TabContent 
//                     activeTab={activeTab}
//                     tabData={tabData}
//                 />
//             </div>

//             {/* configure the message */}
//             <div>
//                 <h5>Configure Message</h5>
//                 <label htmlFor="device-selector">Device: </label>
//                 <Input id="device-selector" type="select" className="api-message-param" onChange={handleDevSelection} required>
//                     {/* initial value of a select option isn't technically selected for some reason */}
//                     {/* <option value="default">Select a device</option> */}
//                     {devList.map((id) => (
//                         <option key={id} value={id}>
//                         {id}
//                         </option>
//                     ))}
//                 </Input>
//                 <br />
//                 <Input type="text" id="send-message-input" defaultValue={msgVal}/>
//                 <small>^reload the page to reset the input box.</small>
//                 <br />
//                 <Button id="send-message-button" onClick={sendToApiSvr} color="primary">Send</Button>
//                 <br />
//                 <List id="message-response-list"></List>
//             </div>
            
//             {/* display the message and the context it resides in */}
//             <div>
//                 <h5>Message History</h5>
//                 <MsgHistoryGrid 
//                     device={selectedDevice} 
//                     after={"2023-10-03T16:45:14.000+00:00"} 
//                     before={"2025-10-03T16:45:14.000+00:00"}
//                 />
//             </div>
//         </div>
//     );
// }