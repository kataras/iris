process.env.NODE_ENV = 'production';

const express = require('express');
const createWorker = require('throng');


createWorker(createWebServer)

function createWebServer() {
  const app = express();

  app.get('/api/values/:id', function (req, res) {
    res.send('value');
  });

  app.listen(5000, function () {
    console.log(
      'Now listening on: http://localhost:5000\nApplication started. Press CTRL+C to shut down.'
    )
  });

}