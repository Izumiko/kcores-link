package main

import (
	"encoding/json"
	"fmt"
	"github.com/tarm/serial"
	"log"
	"net/http"
	"strings"
)

var hub *Hub
var s *serial.Port

var bfstk *BufferStack

func main() {
	bfstk = new(BufferStack)
	// run
	runWEBUIServer()
	runWSServer()
}

type EasyPowerData struct {
	InputVoltage   string
	InputCurrent   string
	InputPower     string
	OutputVoltage  string
	OutputCurrent  string
	OutputPower    string
	IntakeAirTemp  string
	OuttakeAirTemp string
	FanSpeed       string
}

type DataFrame struct {
	OP   string `json:"op"`
	Data string `json:"data"`
}

func OpenSerial(serialPortName string) (*serial.Port, error) {
	c := &serial.Config{Name: serialPortName, Baud: 115200}
	var err error
	s, err = serial.OpenPort(c)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func closeSerial() bool {
	err := s.Close()
	if err != nil {
		log.Fatal(err)
		return false
	}
	return true
}

type BufferStack struct {
	buf []byte
}

func (bs *BufferStack) add(buf []byte) {
	matchTab := map[string]bool{
		"1": true, "2": true, "3": true, "4": true, "5": true, "6": true, "7": true, "8": true, "9": true, "0": true, ",": true, ".": true,
	}

	for _, b := range buf {
		// clean stack
		cb := string(b)
		if cb == "\n" {
			bufcopy := bs.buf
			processsSerialData(bufcopy)
			bs.buf = []byte{} // reset
			continue
		}
		if _, ok := matchTab[cb]; ok {
			bs.buf = append(bs.buf, b)
		}
	}
}

func processsSerialData(buf []byte) {
	fmt.Println(string(buf))
	// parse
	arr := strings.Split(string(buf), ",")
	if len(arr) < 9 {
		return
	}
	var pd EasyPowerData
	pd.InputVoltage = arr[0]
	pd.InputCurrent = arr[1]
	pd.InputPower = arr[2]
	pd.OutputVoltage = arr[3]
	pd.OutputCurrent = arr[4]
	pd.OutputPower = arr[5]
	pd.IntakeAirTemp = arr[6]
	pd.OuttakeAirTemp = arr[7]
	pd.FanSpeed = arr[8]
	// send
	writeIncomeDataToWEB(pd)
}

func ReadSerial() {
	buf := make([]byte, 64)
	_, err := s.Read(buf)
	if err != nil {
		log.Fatal(err)
	}
	bfstk.add(buf)
}

func lisenSerial() {
	for {
		ReadSerial()
	}
}

func runWEBUIServer() {
	fmt.Printf("Listening on localhost:8080 for WEB UI\n")
	go http.ListenAndServe(":8080", http.FileServer(http.Dir("./web-template/")))
}

func runWSServer() {
	// websocket hub
	hub = newHub()
	go hub.run()
	go getDataFromWEB()
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	http.ListenAndServe(":8081", nil)
}

func getDataFromWEB() {
	for {
		select {
		case message := <-hub.broadcast:
			res := &DataFrame{}
			json.Unmarshal(message, &res)
			if res.OP == "connect-serial" {
				_, err := OpenSerial(res.Data)
				if err != nil {
					writeSerialConnectionStatusToWEB(false)
				} else {
					writeSerialConnectionStatusToWEB(true)
					go lisenSerial()
				}
			} else if res.OP == "disconnect-serial" {
				if ok := closeSerial(); ok {
					writeSerialConnectionStatusToWEB(false)
				} else {
					// can not close serial
				}

			}
		}
	}
}

func writeIncomeDataToWEB(d EasyPowerData) {
	// format websocket json info
	tmp := "{\"op\":\"income-data\", \"data\":{\"InputVoltage\":%s,\"InputCurrent\":%s,\"InputPower\":%s,\"OutputVoltage\":%s,\"OutputCurrent\":%s,\"OutputPower\":%s,\"IntakeAirTemp\":%s,\"OuttakeAirTemp\":%s,\"FanSpeed\":%s}}"
	frame := fmt.Sprintf(tmp, d.InputVoltage, d.InputCurrent, d.InputPower, d.OutputVoltage, d.OutputCurrent, d.OutputPower, d.IntakeAirTemp, d.OuttakeAirTemp, d.FanSpeed)
	fmt.Println(frame)
	// send info to websocket data hub
	hub.broadcast <- []byte(frame)
}

func writeSerialConnectionStatusToWEB(connected bool) {
	str := "serial-disconnected"
	if connected {
		str = "serial-connected"
	}
	// format websocket json info
	frame := "{\"op\":\"" + str + "\", \"data\":\"\"}"
	// send info to websocket data hub
	hub.broadcast <- []byte(frame)
}
