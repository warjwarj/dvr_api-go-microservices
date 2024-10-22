import { AgGridReact } from 'ag-grid-react';
import "ag-grid-community/styles/ag-grid.css";

import { Button } from 'reactstrap';

import { useState, useEffect, useMemo } from 'react'
import { fetchMsgHistory } from '../HttpApiConn';
import { formatReqForMsgHistory } from '../Utils';

// columns for table. Could use it in useState
const MsgHistoryColumnDefs = [
  { field: "receivedTime", colId: "receivedTime" },
  { field: "direction", colId: "direction" },
  { field: "message", colId: "message" },
  { field: "packetTime", colId: "packetTime" }
]

// props should expand to (device to display history of), (start time), and (end time)
export function MsgHistoryGrid({ device, after, before }) {

    // Table Data: The data we display and the columns we organise it against.
    const [rowData, setRowData] = useState([]);
    const [colData, setColData] = useState(MsgHistoryColumnDefs)

    // Adjust the state while rendering. TODO - time selection
    const [prevDevice, setPrevDevice] = useState(device)
    const [prevAfter, setPrevAfter] = useState(after)
    const [prevBefore, setPrevBefore] = useState(before)

    // logic to update table upon rerender, if the stae has changed
    if (device !== prevDevice) {
      setPrevDevice(device);
      fetchAndSetRowData()
    }

    function formatMsgHistoryData(data){
      if (data === null || data === undefined) {
        return data
      }
      const formattedData = data[0].MsgHistory.map(msg => ({ 
        receivedTime: new Date(msg.receivedTime).toLocaleString(), 
        packetTime: new Date(msg.packetTime).toLocaleString(),
        direction: msg.direction,
        message: msg.message
      }))
      return formattedData.reverse() // so latest records are at the top
    }

    // send request to API and update the table with the response
    async function fetchAndSetRowData() {
      try {
        const data = await fetchMsgHistory(formatReqForMsgHistory(device, after, before));
        setRowData(formatMsgHistoryData(data));
      } catch (error) {
        console.error('Error fetching data:', error);
      }
    }

    return (
      // wrapping container with theme & size
      <div
        className="ag-theme-quartz"
        style={{ height: 500 }}
      >
        {/* <Button id="send-message-button" onClick={fetchAndSetRowData} color="secondary" size="sm">Reload table</Button> */}
        <AgGridReact
            rowData={rowData}
            columnDefs={MsgHistoryColumnDefs}
            autoSizeStrategy={{type: 'fitCellContents'}}
        />
      </div>
  )
}