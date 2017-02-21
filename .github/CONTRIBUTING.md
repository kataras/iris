If you wanna contribute please submit a PR to one of the [iris-contrib organisation's](https://github.com/iris-contrib) projects.

##### Note that I do not accept pull requests here and that I use the issue tracker for bug reports and proposals only. Please ask questions on the [https://kataras.rocket.chat/channel/iris][Chat] or [http://stackoverflow.com/](http://stackoverflow.com).

## Before Submitting an Issue

First, please do a search in open issues to see if the issue or feature request has already been filed. If there is an issue add your comments to this issue.

The Iris project is distributed across multiple repositories, try to file the issue against the correct repository,

- [Community iris-specific middleware](https://github.com/iris-contrib/middleware/issues?utf8=%E2%9C%93&q=is%3Aopen+is%3Aissue)
- [App reloader and command line tool](https://github.com/kataras/rizla/issues?utf8=%E2%9C%93&q=is%3Aopen+is%3Aissue)
- [View Engine](https://github.com/kataras/go-template/issues?utf8=%E2%9C%93&q=is%3Aopen+is%3Aissue)
- [Sessions](https://github.com/kataras/go-sessions/issues?utf8=%E2%9C%93&q=is%3Aopen+is%3Aissue)
- [About docs.iris-go.com](https://github.com/iris-contrib/gitbook/issues?utf8=%E2%9C%93&q=is%3Aopen+is%3Aissue)
- [About examples](https://github.com/iris-contrib/examples/issues?utf8=%E2%9C%93&q=is%3Aopen+is%3Aissue)
- [Iris main(core)](https://github.com/kataras/iris/issues?utf8=%E2%9C%93&q=is%3Aopen+is%3Aissue)

Before post a new issue do an iris upgrade:

- Delete `$GOPATH/src/gopkg.in/kataras`
- Open shell and execute the command: `go get -u gopkg.in/kataras/iris.v6/iris`
- Try to re-produce the issue
- If the issue still exists, then post the issue with the necessary information.


If the issue is after an upgrade, please read the [HISTORY.md](https://github.com/kataras/iris/blob/v6/HISTORY.md) for any breaking-changes and fixes.

The author answers the same day, perhaps the same hour you post the issue.

It is impossible to notify each user on every change, so to be aware of the framework's changes and be notify about updates
please **star** or **watch** the repository.

If your issue is a closed-personal question then please ask that question on [community chat][Chat].


## Writing Good Bug Reports and Feature Requests

File a single issue per problem and feature request, do not file combo issues.

The more information you can provide, the more likely someone will be successful reproducing the issue and finding a fix. Therefore:

* Provide reproducable steps, what the result of the steps was, and what you would have expected.
* Description of what you expect to happen
* Animated GIFs
* Code that demonstrates the issue
* Version of Iris
* Errors in the Terminal/Console
* When you have glide/godep installed, can you reproduce the issue when starting Iris' station without these?

[Chat]: https://kataras.rocket.chat/channel/iris
