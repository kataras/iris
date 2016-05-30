const stringMessageType = 0;
const intMessageType = 1;
const boolMessageType = 2;
// bytes is missing here for reasons I will explain somewhen
const jsonMessageType = 4;

const prefix = "iris-websocket-message:";
const separator = ";";

const prefixLen = prefix.length;
var separatorLen = separator.length;
var prefixAndSepIdx = prefixLen + separatorLen - 1;
var prefixIdx = prefixLen - 1;
var separatorIdx = separatorLen - 1;

type onConnectFunc = () => void;
type onDisconnectFunc = () => void;
type onNativeMessageFunc = (websocketMessage: string) => void;
type onMessageFunc = (message: any) => void;

class Ws {
    private conn: WebSocket;
    private isReady: boolean;

    // events listeners

    private connectListeners: onConnectFunc[] = [];
    private disconnectListeners: onDisconnectFunc[] = [];
    private nativeMessageListeners: onNativeMessageFunc[] = [];
    private messageListeners: { [event: string]: onMessageFunc[] } = {};

    //

    constructor(endpoint: string, protocols?: string[]) {
        if (!window["WebSocket"]) {
            return;
        }

        if (endpoint.indexOf("ws") == -1) {
            endpoint = "ws://" + endpoint;
        }
        if (protocols != null && protocols.length > 0) {
            this.conn = new WebSocket(endpoint, protocols);
        } else {
            this.conn = new WebSocket(endpoint);
        }

        this.conn.onopen = ((evt: Event): any => {
            this.fireConnect();
            this.isReady = true;
            return null;
        });

        this.conn.onclose = ((evt: Event): any => {
            this.fireDisconnect();
            return null;
        });

        this.conn.onmessage = ((evt: MessageEvent) => {
            this.messageReceivedFromConn(evt);
        });
    }

    //utils

    private isNumber(obj: any): boolean {
        return !isNaN(obj - 0) && obj !== null && obj !== "" && obj !== false;
    }

    private isString(obj: any): boolean {
        return Object.prototype.toString.call(obj) == "[object String]";
    }

    private isBoolean(obj: any): boolean {
        return typeof obj === 'boolean' ||
            (typeof obj === 'object' && typeof obj.valueOf() === 'boolean');
    }

    private isJSON(obj: any): boolean {
        try {
            JSON.parse(obj);
        } catch (e) {
            return false;
        }
        return true;
    }

    //

    // messages
    private _msg(event: string, messageType: number, dataMessage: string): string {

        return prefix + event + separator + String(messageType) + separator + dataMessage;
    }

    private encodeMessage(event: string, data: any): string {
        let m = "";
        let t = 0;
        if (this.isNumber(data)) {
            t = intMessageType;
            m = data.toString();
        } else if (this.isBoolean(data)) {
            t = boolMessageType;
            m = data.toString();
        } else if (this.isString(data)) {
            t = stringMessageType;
            m = data.toString();
        } else if (this.isJSON(data)) {
            //propably json-object
            t = jsonMessageType;
            m = JSON.stringify(data);
        } else {
            console.log("Invalid");
        }

        return this._msg(event, t, m);
    }

    private decodeMessage<T>(event: string, websocketMessage: string): T | any {
        //iris-websocket-message;user;4;themarshaledstringfromajsonstruct
        let skipLen = prefixLen + separatorLen + event.length + 2;
        if (websocketMessage.length < skipLen + 1) {
            return null;
        }
        let messageType = parseInt(websocketMessage.charAt(skipLen - 2));
        let theMessage = websocketMessage.substring(skipLen, websocketMessage.length);
        if (messageType == intMessageType) {
            return parseInt(theMessage);
        } else if (messageType == boolMessageType) {
            return Boolean(theMessage);
        } else if (messageType == stringMessageType) {
            return theMessage;
        } else if (messageType == jsonMessageType) {
            return JSON.parse(theMessage);
        } else {
            return null; // invalid
        }
    }

    private getCustomEvent(websocketMessage: string): string {
        if (websocketMessage.length < prefixAndSepIdx) {
            return "";
        }
        let s = websocketMessage.substring(prefixAndSepIdx, websocketMessage.length);
        let evt = s.substring(0, s.indexOf(separator));

        return evt;
    }

    private getCustomMessage(event: string, websocketMessage: string): string {
        let eventIdx = websocketMessage.indexOf(event + separator);
        let s = websocketMessage.substring(eventIdx + event.length + separator.length+2, websocketMessage.length);
        return s;
    }

    //

    // Ws Events

    // messageReceivedFromConn this is the func which decides
    // if it's a native websocket message or a custom iris-ws message
    // if native message then calls the fireNativeMessage
    // else calls the fireMessage
    //
    // remember Iris gives you the freedom of native websocket messages if you don't want to use this client side at all.
    private messageReceivedFromConn(evt: MessageEvent): void {
        //check if iris-ws message
        let message = <string>evt.data;
        if (message.indexOf(prefix) != -1) {
            let event = this.getCustomEvent(message);
            if (event != "") {
                // it's a custom message
                this.fireMessage(event, this.getCustomMessage(event, message));
                return;
            }
        }

        // it's a native websocket message
        this.fireNativeMessage(message);
    }

    OnConnect(fn: onConnectFunc): void {
        if (this.isReady) {
            fn();
        }
        this.connectListeners.push(fn);
    }

    fireConnect(): void {
        for (let i = 0; i < this.connectListeners.length; i++) {
            this.connectListeners[i]();
        }
    }

    OnDisconnect(fn: onDisconnectFunc): void {
        this.disconnectListeners.push(fn);
    }

    fireDisconnect(): void {
        for (let i = 0; i < this.disconnectListeners.length; i++) {
            this.disconnectListeners[i]();
        }
    }

    OnMessage(cb: onNativeMessageFunc): void {
        this.nativeMessageListeners.push(cb);
    }

    fireNativeMessage(websocketMessage: string): void {
        for (let i = 0; i < this.nativeMessageListeners.length; i++) {
            this.nativeMessageListeners[i](websocketMessage);
        }
    }

    On(event: string, cb: onMessageFunc): void {
        if (this.messageListeners[event] == null || this.messageListeners[event] == undefined) {
            this.messageListeners[event] = [];
        }
        this.messageListeners[event].push(cb);
    }

    fireMessage(event: string, message: any): void {
        for (let key in this.messageListeners) {
            if (this.messageListeners.hasOwnProperty(key)) {
                if (key == event) {
                    for (let i = 0; i < this.messageListeners[key].length; i++) {
                        this.messageListeners[key][i](message);
                    }
                }
            }
        }
    }


    //

    // Ws Actions

    Disconnect(): void {
        this.conn.close();
    }

    // EmitMessage sends a native websocket message
    EmitMessage(websocketMessage: string): void {
        this.conn.send(websocketMessage);
    }

    // Emit sends an iris-custom websocket message
    Emit(event: string, data: any): void {
        let messageStr = this.encodeMessage(event, data);
        this.EmitMessage(messageStr);
    }

    //

}

// node-modules export {Ws};