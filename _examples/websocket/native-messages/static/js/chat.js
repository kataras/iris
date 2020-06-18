var messageTxt = document.getElementById("messageTxt");
var messages = document.getElementById("messages");
var sendBtn = document.getElementById("sendBtn")

w = new WebSocket("ws://" + HOST + "/my_endpoint");
w.onopen = function () {
	console.log("Websocket connection enstablished");
};

w.onclose = function () {
	appendMessage("<div><center><h3>Disconnected</h3></center></div>");
};
w.onmessage = function (message) {
	appendMessage("<div>" + message.data + "</div>");
};

sendBtn.onclick = function () {
	myText = messageTxt.value;
	messageTxt.value = "";

	appendMessage("<div style='color: red'> me: " + myText + "</div>");
	w.send(myText);
};

messageTxt.addEventListener("keyup", function (e) {
	if (e.keyCode === 13) {
		e.preventDefault();

		sendBtn.click();
	}
});

function appendMessage(messageDivHTML) {
	messages.insertAdjacentHTML('afterbegin', messageDivHTML);
}
