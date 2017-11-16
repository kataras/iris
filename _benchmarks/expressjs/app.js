process.env.NODE_ENV = 'production';

const express = require('express');
const app = express();

app.get('/api/values/:id', function (req, res) {
  res.send('value');
});

app.listen(5000, function () {
  console.log(
    'Now listening on: http://localhost:5000\nApplication started. Press CTRL+C to shut down.'
  )
});