# A Todo MVC Application using Iris and Vue.js

## The Tools

Programming Languages are just tools for us, but we need a safe, fast and “cross-platform” programming language to power our service.

[Go](https://golang.org) is a [rapidly growing](https://www.tiobe.com/tiobe-index/) open source programming language designed for building simple, fast, and reliable software. Take a look [here](https://github.com/golang/go/wiki/GoUsers) which great companies use Go to power their services.

### Install the Go Programming Language

Extensive information about downloading & installing Go can be found [here](https://golang.org/dl/).

[![](https://i3.ytimg.com/vi/9x-pG3lvLi0/hqdefault.jpg)](https://youtu.be/9x-pG3lvLi0)

> Maybe [Windows](https://www.youtube.com/watch?v=WT5mTznJBS0) or [Mac OS X](https://www.youtube.com/watch?v=5qI8z_lB5Lw) user?

> The article does not contain an introduction to the language itself, if you’re a newcomer I recommend you to bookmark this article, [learn](https://github.com/golang/go/wiki/Learn) the language’s fundamentals and come back later on.

## The Dependencies

Many articles have been written, in the past, that lead developers not to use a web framework because they are useless and "bad". I have to tell you that there is no such thing, it always depends on the (web) framework that you’re going to use. At production environment,  we don’t have the time or the experience to code everything that we wanna use in the applications, and if we could are we sure that we can do better and safely than others? In short term: **Good frameworks are helpful tools for any developer, company or startup and "bad" frameworks are waste of time, crystal clear.**

You’ll need only two dependencies:

1. The Iris Web Framework, for our server-side requirements. Can be found [here](https://github.com/kataras/iris)
2. Vue.js, for our client-side requirements. Download it from [here](https://vuejs.org/)

> If you have Go already installed then just execute `go get -u github.com/kataras/iris` to install the Iris Web Framework.

## Start

If we are all in the same page, it’s time to learn how we can create a live todo application that will be easy to deploy and extend even more!

We're going to use a vue.js todo application which uses browser'
s local storage and doesn't have any user-specified features like live sync between browser's tabs, you can find the original version inside the vue's [docs](https://vuejs.org/v2/examples/todomvc.html).

### The client-side (vue.js)

### The server-side (iris)

## References

https://vuejs.org/v2/examples/todomvc.html (using browser's local storage)