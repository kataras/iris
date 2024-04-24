(function () {
  var events = {
    default: {
      _OnNamespaceConnected: function (ns, msg) {
        ns.joinRoom(PAGE_SOURCE);
      },
      _OnNamespaceDisconnect: function (ns, msg) {
        document.getElementById("online_views").innerHTML = "you've been disconnected";
      },
      onNewVisit: function (ns, msg) {
        var text = "1 online view";
        var onlineViews = Number(msg.Body);
        if (onlineViews > 1) {
          text = onlineViews + " online views";
        }
        document.getElementById("online_views").innerHTML = text;
      }
    }
  };

  neffos.dial("ws://localhost:8080/my_endpoint", events).then(function (client) {
    client.connect("default");
  });
})();
