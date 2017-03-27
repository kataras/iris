# Iris Path Expression Language (_future)


## Ideas & Goals 

- Optional.
- No Breaking Changes.
- No performance cost if not used.
- Can convert a path for the existing routers, if no router is being used, then it will use its own, new, router.
- 4+1 basic parameter types:  `string`, `int`, `alphabetical`, `file`, `path` (file with any number of slashes), based on regexp.
- Each type has unlimited functions of its own, they should be able to be overriden.
- Give devs the ability to parse their function's arguments before use them and return a func which is the validator.
- Function will be a generic type(`interface{}`) in order to devs be able to use any type without boilerplate code for conversions,
can be done using reflection and reflect.Call, on .Boot time to parse the function automatically, and keep the returning validator function (already tested and worked).
- The `string` will be the default if dev use functions to the named path parameter but missing a type.
- If a type doesnt't contains a function of its own, then it will use the `string`'s, so `string` will contain global-use functions too. 

## Preview

`/api/users/{id:int min(1)}/posts`

```go
minValidator := func(min int) func(string) bool {
    return func(paramValue string) bool {
       	paramValueInt, err := strconv.Atoi(paramValue)
		if err != nil {
			return false
		}
        if paramValueInt < min {
            return false
        }
        return true
    }
}

app := iris.New()
app.Int.Set("min", minValidator)
```

`/api/{version:string len(2) isVersion()}`

```go
isVersionStrValidator := func() func(string) bool {
    versions := []string{"v1","v2"}
    return func(paramValue string) bool {
        for _, s := range versions {
            if s == paramValue {
                return true
            }
        }
        return false
    }
}

lenStrValidator := func(i int) func(string) bool {
    if i <= 0 {
        i = 1
    }
    return func(paramValue string) bool {
       return len(paramValue) != i
    }
}


app := iris.New()
app.String.Set("isVersion", isVersionStrValidator)
app.String.Set("len", lenStrValidator)
```

`/uploads/{fullpath:path contains(.) else 403}`

```go
[...]

[...]
```

`/api/validate/year/{year:int range(1970,2017) else 500}`

```go
[...]

[...]
```

## Resources
- [Lexical analysis](https://en.wikipedia.org/wiki/Lexical_analysis) **necessary**
- [Top-down parsing](https://en.wikipedia.org/wiki/Top-down_parsing) **necessary**
- [Recursive descent parser](https://en.wikipedia.org/wiki/Recursive_descent_parser) **basic, continue to the rest after**
- [Handwritten Parsers & Lexers in Go](https://blog.gopheracademy.com/advent-2014/parsers-lexers/) **very good**
- [Creating a VM / Compiler Episode 1: Bytecode VM](https://www.youtube.com/watch?v=DUNkdl0Jhgs) **nice to watch**
- [So you want to write an interpreter?](https://www.youtube.com/watch?v=LCslqgM48D4) **watch it, continue to the rest later on**
- [Writing a Lexer and Parser in Go - Part 1](http://adampresley.github.io/2015/04/12/writing-a-lexer-and-parser-in-go-part-1.html) **a different approach using the strategy pattern, not for production use in my opinion**
- [Writing a Lexer and Parser in Go - Part 2](http://adampresley.github.io/2015/05/12/writing-a-lexer-and-parser-in-go-part-2.html)
- [Writing a Lexer and Parser in Go - Part 3](http://adampresley.github.io/2015/06/01/writing-a-lexer-and-parser-in-go-part-3.html)
- [Writing An Interpreter In Go](https://www.amazon.com/Writing-Interpreter-Go-Thorsten-Ball/dp/300055808X) **I recommend this book: suitable for both experienced and novice developers**


<!-- author's notes:

When finish, I should write an article for new Gophers,  based on all of that I have read the last months on this subject.
It will help a lot new developers looking for these things.

Also, don't push commits to the _future folder for each change, commit every 2-3 days is enough.

-->
