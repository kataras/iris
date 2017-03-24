(function() {
  var socket = new Ws("ws://localhost:8080/my_endpoint");

  socket.OnConnect(function () {
      socket.Emit("watch", PAGE_SOURCE);
  });


  socket.On("watch", function (onlineViews) {
      var text = "1 online view";
      if (onlineViews > 1) {
          text = onlineViews + " online views";
      }
      document.getElementById("online_views").innerHTML = text;
  });

  socket.OnDisconnect(function () {
    document.getElementById("online_views").innerHTML = "you've been disconnected";
  });

})();
