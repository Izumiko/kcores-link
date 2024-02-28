package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"go.bug.st/serial"
)

//go:embed static
var staticContent embed.FS

var hub *Hub
var s serial.Port
var serialName string

var bfstk *BufferStack

var verbose bool

func main() {
	verbose = false
	args := os.Args[1:]
	if len(args) > 0 {
		if args[0] == "-v" {
			verbose = true
		}
	}

	bfstk = new(BufferStack)

	fmt.Printf("Listening on localhost:8080 for WEB UI\n")
	r := mux.NewRouter()

	// Handle WebSocket connection first
	hub = newHub()
	go hub.run()
	go getDataFromWEB()
	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	// Serve static files from embedded content
	view, _ := fs.Sub(staticContent, "static")
	fileServer := http.FileServer(http.FS(view))
	r.PathPrefix("/").Handler(fileServer)

	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type EasyPowerData struct {
	InputVoltage   float32 `json:"InputVoltage"`
	InputCurrent   float32 `json:"InputCurrent"`
	InputPower     float32 `json:"InputPower"`
	OutputVoltage  float32 `json:"OutputVoltage"`
	OutputCurrent  float32 `json:"OutputCurrent"`
	OutputPower    float32 `json:"OutputPower"`
	IntakeAirTemp  float32 `json:"IntakeAirTemp"`
	OuttakeAirTemp float32 `json:"OuttakeAirTemp"`
	FanSpeed       float32 `json:"FanSpeed"`
}

type DataFrame struct {
	OP   string `json:"op"`
	Data string `json:"data"`
}

type StatusDataFrame struct {
	OP         string        `json:"op"`
	SerialName string        `json:"serialName"`
	Data       EasyPowerData `json:"data"`
}

type SerialListDataFrame struct {
	OP   string   `json:"op"`
	Data []string `json:"data"`
}

func OpenSerial(serialPortName string) (serial.Port, error) {
	mode := &serial.Mode{
		BaudRate: 115200,
	}
	var err error
	s, err = serial.Open(serialPortName, mode)
	if err != nil {
		return nil, err
	}
	serialName = serialPortName

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

func parseFloat32(s string) float32 {
	f, _ := strconv.ParseFloat(s, 32)
	return float32(f)
}

func processsSerialData(buf []byte) {
	if verbose {
		fmt.Println(string(buf))
	}
	// parse
	arr := strings.Split(string(buf), ",")
	if len(arr) < 9 {
		return
	}
	var pd EasyPowerData
	pd.InputVoltage = parseFloat32(arr[0])
	pd.InputCurrent = parseFloat32(arr[1])
	pd.InputPower = parseFloat32(arr[2])
	pd.OutputVoltage = parseFloat32(arr[3])
	pd.OutputCurrent = parseFloat32(arr[4])
	pd.OutputPower = parseFloat32(arr[5])
	pd.IntakeAirTemp = parseFloat32(arr[6])
	pd.OuttakeAirTemp = parseFloat32(arr[7])
	pd.FanSpeed = parseFloat32(arr[8])
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

func listenSerial() {
	for {
		ReadSerial()
	}
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
					go listenSerial()
				}
			} else if res.OP == "disconnect-serial" {
				if ok := closeSerial(); ok {
					writeSerialConnectionStatusToWEB(false)
				} else {
					// can not close serial
				}
			} else if res.OP == "list-serial" {
				writeSerialListToWEB()
			}
		}
	}
}

func writeIncomeDataToWEB(d EasyPowerData) {
	// format websocket json info
	var info StatusDataFrame
	info.OP = "income-data"
	info.SerialName = serialName
	info.Data = d
	frame, _ := json.Marshal(info)
	if verbose {
		fmt.Println(string(frame))
	}
	// send info to websocket data hub
	hub.broadcast <- frame
}

func writeSerialConnectionStatusToWEB(connected bool) {
	op := "serial-disconnected"
	d := ""
	if connected {
		op = "serial-connected"
		d = serialName
	}
	// format websocket json info
	var info DataFrame
	info.OP = op
	info.Data = d
	frame, _ := json.Marshal(info)
	// send info to websocket data hub
	hub.broadcast <- frame
}

func writeSerialListToWEB() {
	var info SerialListDataFrame
	info.OP = "serial-list"
	ports, err := serial.GetPortsList()
	if err != nil {
		log.Fatal(err)
	}
	info.Data = ports
	// format websocket json info
	frame, _ := json.Marshal(info)
	// send info to websocket data hub
	hub.broadcast <- frame
}
