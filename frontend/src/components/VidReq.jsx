import React, { useEffect, useRef, useState } from 'react';
import { Input } from 'reactstrap';

import { formatDateTimeDVRFormat } from '../Utils';

// PROPS: 
// active: is the vidreq widget active? If so we set msg value to default vidreq.
// setCurrentMsg: change the message value in the input widget
export default function VidReq({ setMsgVal }) {

    // $VIDEO;[DeviceID];[type];[camera];[start];[time length]<CR>
    const [ vidreq, setVidreq ] = useState(["$VIDEO", "0", "all", "0", "0", "0"])

    useEffect(() => {
        setMsgVal(vidreq.join(';'))
    }, [vidreq])

    const handleChange = (event) => {
        // new arr
        const temp = [...vidreq];
        switch (event.target.id){
            case "vidreq-datetime-input":
                temp[4] = formatDateTimeDVRFormat(event.target.value);
                break;
            case "vidreq-length-input":
                temp[5] = event.target.value;
                break;
            case "vidreq-chnum-input":
                temp[3] = event.target.value;
                break;
            default:
                console.log("unrecognised event target id in interpretInputValue")
        }
        setVidreq(temp);
    };

    return (
        <>
            <label htmlFor="vidreq-datetime-input">Start Time: </label>
            <Input 
                type="datetime-local" 
                id="vidreq-datetime-input"
                className="api-message-param" 
                onChange={handleChange} 
                required 
            />
            <br />
            <label htmlFor="vidreq-length-input">Length: </label>
            <Input 
                type="number" 
                id="vidreq-length-input" 
                className="api-message-param" 
                min="0" 
                max="300" 
                onChange={handleChange} 
                required 
            />
            <br />
            <label htmlFor="vidreq-chnum-input">Channel: </label>
            <Input 
                type="number" 
                id="vidreq-chnum-input" 
                className="api-message-param" 
                min="0" 
                max="8" 
                onChange={handleChange} 
                required 
            />
        </>
    );
}