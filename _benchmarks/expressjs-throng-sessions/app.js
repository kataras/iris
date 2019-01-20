process.env.NODE_ENV = 'production';

const express = require('express');
const app = express();
const session = require('express-session');
const createWorker = require('throng');
const MemoryStore = require('session-memory-store')(session);

createWorker(createWebServer)

function createWebServer() {
  // Use the session middleware with the memory-store as
  // recommended for production local use, like with iris and netcore.
  app.use(session({ store: new MemoryStore({expires: 1*60, checkperiod: 1*60}), secret: '.cookiesession.id', resave: true, saveUninitialized: false, cookie: { secure: true, maxAge: 60000 } }));
  
  app.get('/setget', function (req, res) {
    req.session.key = 'value';

    var value = req.session.key;
    if (value == '') {
      res.send('NOT_OK');
      return;
    }

    res.send(value);
  });

  app.listen(5000, function () {
    console.log(
      'Now listening on: http://localhost:5000\nApplication started. Press CTRL+C to shut down.'
    )
  });
}