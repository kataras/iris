const neffos = require('neffos.js');
const stdin = process.openStdin();

const wsURL = "ws://localhost:8080/echo";

async function runExample() {
  try {
    const conn = await neffos.dial(wsURL, {
      default: { // "default" namespace.
        _OnNamespaceConnected: function (nsConn, msg) {
          console.log("connected to namespace: " + msg.Namespace);
        },
        _OnNamespaceDisconnect: function (nsConn, msg) {
          console.log("disconnected from namespace: " + msg.Namespace);
        },
        chat: function (nsConn, msg) { // "chat" event.
          console.log(msg.Body);
        }
      }
    });

    const nsConn = await conn.connect("default");
    nsConn.emit("chat", "Hello from Nodejs client side!");

    stdin.addListener("data", function (data) {
      const text = data.toString().trim();
      nsConn.emit("chat", text);
    });

  } catch (err) {
    console.error(err);
  }
}

runExample();
