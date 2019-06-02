# Browserify example

```sh
$ npm install --only=dev # install browserify from the devDependencies.
$ npm run-script build # browserify and minify the `app.js` into `bundle.js`.
$ cd ../ && go run server.go # start the neffos server.
```

> make sure that you have [golang](https://golang.org/dl) installed to run and edit the neffos (server-side).

That's all, now navigate to <http://localhost:8080/browserify>.
