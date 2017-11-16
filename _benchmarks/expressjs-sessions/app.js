process.env.NODE_ENV = 'production';

const express = require('express');
const app = express();
const session = require('express-session');

// Use the session middleware
app.use(session({ secret: '.cookiesession.id', resave: true, saveUninitialized: false, cookie: { secure: true, maxAge: 60000 } }));

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