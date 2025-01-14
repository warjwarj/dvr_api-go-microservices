<h1>dvr_api</h1>

<h2>Documentation</h2>

Send and receive messages to and from devices, view history of messages sent to and from devices.

<h3>HTTP API</h3>

<h2>POST Messsage History: ?reqType=MessageHistory</h2>

Send JSON in an HTTP GET request to get message history according to the parameters.<br>
Will return  message history for devices in the list, where the <strong>packet time</strong> of the element is between the two stated times.<br>

<h4>REQUEST - Example HTTP POST request body to get message history</h4>
{
  "after": "2023-10-03T16:45:14.000+00:00",
  "before": "2025-10-03T16:45:14.000+00:00",
  "devices": [
    "123456",
    "222",
    "444"
  ]
}
<br>

<h4>RESPONSE - Example of a response to a message history request</h4>
[
  {
    "DeviceId": "123456",
    "MsgHistory": [
      {
        "direction": "to",
        "message": "$VIDEO;123456;20240817-123504;pokpok\r",
        "packeTime": "2024-08-17T12:35:04Z",
        "receivedTime": "2024-08-24T20:43:21.29Z"
      },
      {
        "direction": "to",
        "message": "$VIDEO;123456;20240817-123504;pokpok\r",
        "packetTime": "2024-08-17T12:35:04Z",
        "receivedTime": "2024-08-24T20:43:26.927Z"
      },
    ]
  }
]
<br>

<h4>REQUEST connected devices: ?reqType=GetConnectedDevices</h4>

Send a get request to get a list of devices connected to the server.<br>
Will return a JSON object with the key "ConnectedDevices" addressing the object field containing them, an array of strings.<br>

<h4>RESPONSE - list of connected devices</h4>
{"ConnectedDevices":["111"]}

<br><br><br>

<h3>WS API - Live Messaging</h3>

Connect to the websocket endpoint with the subprotocol 'dvr_api'.<br>
Send the following JSON fields, seperate or in the same JSON object.<br>

<h4>REQUEST - Example websocket API request to send, receive messages</h4>
{
  "messages": ["$VIDEO;123456;all;4;20231003-164514;5", "$VIDEO;654321;all;4;20231003-164514;5"],
  "subscriptions": ["123456", "654321"]
}
<br>

<h4>RESPONSE - Example message forwarded from a device subscribed to</h4>
{
  "recvdTime": "2024-08-26T12:17:37.2952618+01:00",
  "packetTime": "2024-08-17T12:35:04Z",
  "message": "$VIDEO;123456;20240817-123504;pokpok\r",
  "direction": "from"
}
<br><br>

The "messages" field will send each message to the pertinent device, according to the 'device' field (technically defined as, having split the message into sections by the semicolons, the first section entirely made up of numbers.)<br>

The "subscriptions" field will track your subscriptions each time you send the field. the server will forward every message that the devices in the list send, to you the subscriber. Note that if you want to receive the response to any message sent, unless the DVR delays the response by a second or do, you’ll need to send the subscription request before you send the messages.<br>

The "getConnectedDevices" field will, immidiately after you send the field, send a response as a JSON object with one field, "connectedDevicesList", the key to a value containing a list of all connected devices which you can then subscribe to and send messages to.<br>
