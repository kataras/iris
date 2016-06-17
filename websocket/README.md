# Package information

This package is new and unique, if you notice a bug or issue [post it here](https://github.com/kataras/iris/issues).

# How to use

[E-Book section](https://kataras.gitbooks.io/iris/content/package-websocket.html)


## Notes

On **OSX + Safari**, we had an issue which is **fixed** now. BUT by the browser's Engine Design the socket is not closed until the whole browser window is closed,
so the **connection.OnDisconnect** event will fire when the user closes the **window browser**, **not just the browser's tab**.

- Relative issue: https://github.com/kataras/iris/issues/175
