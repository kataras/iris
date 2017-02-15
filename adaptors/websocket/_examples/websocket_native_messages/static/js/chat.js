var messageTxt;
var messages;

$(function () {

	messageTxt = $("#messageTxt");
	messages = $("#messages");


	w = new WebSocket("ws://" + HOST + "/my_endpoint");
	w.onopen = function () {
		console.log("Websocket connection enstablished");
	};

	w.onclose = function () {
		appendMessage($("<div><center><h3>Disconnected</h3></center></div>"));
	};
	w.onmessage = function(message){
		appendMessage($("<div>" + message.data + "</div>"));
	};


	$("#sendBtn").click(function () {
		w.send(messageTxt.val().toString());
		messageTxt.val("");
	});

})


function appendMessage(messageDiv) {
    var theDiv = messages[0];
    var doScroll = theDiv.scrollTop == theDiv.scrollHeight - theDiv.clientHeight;
    messageDiv.appendTo(messages);
    if (doScroll) {
        theDiv.scrollTop = theDiv.scrollHeight - theDiv.clientHeight;
    }
}
