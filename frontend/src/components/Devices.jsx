// external
import React, { useState, useEffect } from 'react';
import { Button, List, Input } from 'reactstrap';

// js
import WsApiConn from '../WsApiConn';
import { getCurrentTime, formatDateTimeDVRFormat } from '../Utils';

// components
import { TabContent, TabButtons } from './Tabs';
import VidReq from './VidReq';
import { MsgHistoryGrid } from './MsgHistoryGrid';
import { fetchConnectedDevices } from '../HttpApiConn';

export default function Devices() {

    //~~~~~~~~~~~~~~~~~~~~~~~~~
    // HOOKS
    //~~~~~~~~~~~~~~~~~~~~~~~~~

    // states
    const [ msgVal, setMsgVal ] = useState("")
    const [ devList, setDevList ] = useState(["Select Device"]);
    const [ selectedDevice, setSelectedDevice ] = useState("");
    const [ activeTab, setActiveTab ] = useState(0);

    // PAGE LOAD - set callbacks, populate the list of connected devices, subscribe to messages
    useEffect(() => {
        const fetchAndSetConnectedDevices = async () => {
            const data = await fetchConnectedDevices()
            setDevList(data.ConnectedDevices)
        }
        // set what we want to do with received data
        WsApiConn.setReceiveCallback(Devices_WsApiConnectionCallback)
        // get the connected devices list from the server
        fetchAndSetConnectedDevices()
    }, []);

    useEffect(() => {
        const sendSubReq = () => {
            // Subscribe to receive all messages from all devices connected to the server
            apiReq.Subscriptions = [selectedDevice]
            WsApiConn.apiConnection.send(JSON.stringify(apiReq));
        } 
        if (selectedDevice != null && WsApiConn.apiConnection.readyState === WebSocket.OPEN) {
            sendSubReq()
        }

    }, [selectedDevice]);

    //~~~~~~~~~~~~~~~~~~~~~~~~~
    // PAGE VARIABLES
    //~~~~~~~~~~~~~~~~~~~~~~~~~

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

    // reusable request object to keep the subscription field the same
    const apiReq = {
        "Messages": [],
        "Subscriptions": []
    }

    
    //~~~~~~~~~~~~~~~~~~~~~~~~~
    // CALLBACKS/EVENT HANDLERS
    //~~~~~~~~~~~~~~~~~~~~~~~~~

    // WS API CALLBACK - fired upon every event we receive from the websocket API connection
    const  Devices_WsApiConnectionCallback = (event) => {
        // parse the message we've received over the WS connection
        const payload = JSON.parse(event.data)
        console.log(payload)
        if ("Message" in payload) {
            addMessageToLog(payload.Message)
        }
    }

    // record a device message notification we've received from the server
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
        apiReq.Messages = [message]
        WsApiConn.apiConnection.send(JSON.stringify(apiReq))
    }

    //~~~~~~~~~~~~~~~~~~~~~~~~~
    // JSX
    //~~~~~~~~~~~~~~~~~~~~~~~~~

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
                <Input id="device-selector" type="select" className="api-message-param" onChange={handleDevSelection} value={selectedDevice} required>
                    {/* initial value of a select option isn't technically selected for some reason */}
                    <option value="">Select a device</option>
                    {devList && devList.length > 0 ? (
                        devList.map((id) => (
                            <option key={id} value={id}>
                                {id}
                            </option>
                        ))
                    ) : (
                        <option disabled>No devices are connected</option>
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