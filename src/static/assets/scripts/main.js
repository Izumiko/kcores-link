import { CountUp } from './countUp.min.js';

const options = {
    // startVal: 0,      // 开始值
    // decimalPlaces: 0, // 小数位
    // duration: 2,      // 持续时间
    useEasing: true,  // 使用缓和
    separator: '',    // 分隔器(千位分隔符,默认为',')
    decimal: '.',     // 十进制(小数点符号,默认为 '.')
    prefix: '',       // 字首(数字的前缀,根据需要可设为 $,¥,￥ 等)
    suffix: ''        // 后缀(数字的后缀 ,根据需要可设为 元,个,美元 等) 
}

const appendServerData = (serverData) => {
    new CountUp('input-voltage', serverData.data.InputVoltage, { startVal: parseFloat(document.getElementById('input-voltage').innerHTML), decimalPlaces: 1, duration: 2, ...options }).start()
    new CountUp('input-current', serverData.data.InputCurrent, { startVal: parseFloat(document.getElementById('input-current').innerHTML), decimalPlaces: 1, duration: 2, ...options }).start()
    new CountUp('input-power', serverData.data.InputPower, { startVal: parseFloat(document.getElementById('input-power').innerHTML), decimalPlaces: 1, duration: 2, ...options }).start()
    new CountUp('output-voltage', serverData.data.OutputVoltage, { startVal: parseFloat(document.getElementById('output-voltage').innerHTML), decimalPlaces: 1, duration: 2, ...options }).start()
    new CountUp('output-current', serverData.data.OutputCurrent, { startVal: parseFloat(document.getElementById('output-current').innerHTML), decimalPlaces: 1, duration: 2, ...options }).start()
    new CountUp('output-power', serverData.data.OutputPower, { startVal: parseFloat(document.getElementById('output-power').innerHTML), decimalPlaces: 1, duration: 2, ...options }).start()
    new CountUp('intake-air-temp', serverData.data.IntakeAirTemp, { startVal: parseFloat(document.getElementById('intake-air-temp').innerHTML), decimalPlaces: 2, duration: 2, ...options }).start()
    new CountUp('outtake-air-temp', serverData.data.OuttakeAirTemp, { startVal: parseFloat(document.getElementById('outtake-air-temp').innerHTML), decimalPlaces: 2, duration: 2, ...options }).start()
    new CountUp('fan-speed', serverData.data.FanSpeed, { startVal: parseFloat(document.getElementById('fan-speed').innerHTML), decimalPlaces: 0, duration: 2, ...options }).start()
}

window.onload = () => {
    // load websocket
    let conn;
    let connected = false;
    if (window["WebSocket"]) {
        const url = window.origin.replace(/^http/, 'ws') + '/ws';
        conn = new WebSocket(url);
        conn.onclose = (evt) => {
            serialDisconnected()
        };
        conn.onmessage = (evt) => {
            const messages = evt.data.split('\n');
            for (let i = 0; i < messages.length; i++) {
                const receivedServerData = JSON.parse(messages[i]);

                // op switch
                if (receivedServerData.op === "income-data") {
                    appendServerData(receivedServerData);
                } else if (receivedServerData.op === "serial-connected") {
                    serialConnected();
                } else if (receivedServerData.op === "serial-disconnected") {
                    serialDisconnected();
                }
            }
        };
    } else {
        const item = document.createElement("div");
        item.innerHTML = "<b>Your browser does not support WebSockets.</b>";
        document.body.prepend(item);
    }

    // add listeners
    let serialPortName;
    document.getElementById("connect-button").onclick = () => {
        if (!connected) {
            connectSerial()
        } else {
            disconnectSerial()
        }
    };
    document.getElementById('connect-input-box').onclick = () => {
        serialPortName = document.getElementById('connect-input-box').value;
        if (!serialPortName || serialPortName === "") {
            resetConnectButton()
            return false;
        }
    };

    const connectSerial = () => {
        if (!conn) {
            return false;
        }
        serialPortName = document.getElementById('connect-input-box').value;
        if (!serialPortName || serialPortName === "") {
            pleaseInputSerialPortName();
            return false;
        }
        conn.send("{\"op\":\"connect-serial\", \"data\":\"" + serialPortName + "\"}");
        return true;
    }
    const disconnectSerial = () => {
        if (!conn) {
            return false;
        }
        if (!document.getElementById('connect-input-box').value) {
            return false;
        }
        serialPortName = document.getElementById('connect-input-box').value;
        conn.send("{\"op\":\"disconnect-serial\", \"data\":\"\"}");
        return true;
    }
    const resetConnectButton = () => {
        const btn = document.getElementById("connect-button");
        btn.innerText = "点击连接";
    };
    const pleaseInputSerialPortName = () => {
        document.getElementById("connect-button").innerText = "请输入串口名称";
    }
    const serialConnected = () => {
        const btn = document.getElementById("connect-button");
        btn.innerText = "已连接";
        btn.className = "content-button-text-connected"
    }
    const serialDisconnected = () => {
        const btn = document.getElementById("connect-button");
        btn.innerText = "已断开";
        btn.className = "content-button-text"
    }
}