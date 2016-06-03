## Package information

This package is the command line tool for  [../](https://github.com/kataras/iris).


## Install
Current version: 0.0.3
```sh

go get -u github.com/kataras/iris/iris

```

## Usage


```sh
$ iris [command] [-flags]
```

> Note that you must have $GOPATH/bin to your $PATH system/environment variable.


## Create


**The create command** creates for you a start project in a directory


```sh
iris create -t basic -d ./
```

Will create the  [basic](https://github.com/iris-contrib/iris-command-assets/tree/master/basic) sample package to the current working directory and run the app.

```sh
iris create -t static  -d ./
```

Will create the [static](https://github.com/iris-contrib/iris-command-assets/tree/master/static) sample package to the current working directory and run the app.


The default

```sh
iris create
```

Will create the basic sample package to the current working directory and run the app.

```sh
iris create -d C:\Users\kataras\Desktop\test1
```

Will create the basic sample package to the C:\Users\kataras\Desktop\test1 folder and run the app.


## Version

```sh
iris version
```

Will print the current iris' installed version to your machine

## TODO

- [ ] Add more templates
