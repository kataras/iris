# Specs for contributors

(a) *implementation type*: 
     `Processor`, or `Joiner`, or `Divider`, or `Validator`.

(b) *implementation*:
     a function or a variable which returns a value of an *implementation type*.

(c) *implementation description*:
     the Name of a function which returns an *implementation*. It describes what it does, as we do normally as programmers.


## Naming

### Source Files

Generally speaking, you can name the source files **based on your experience** and if you are not sure, just open a new [issue](https://github.com/esemplastic/unis/issues/new), before PR, in order to find a solution and decide it all together. 

We have only **one basic rule which we are not allowed to ignore**:
A new source file which holds an *implementation type* should be named with the suffix of its specific `implementation type`, i.e `xxxx_processor.go`, `xxxx_joiner.go`, `xxxx_divider.go`, `xxxx_validator.go`. Why? Because we may have almost same implementation but differnet usage in different *implementation types*, example: `joiner_processor.go: is a Processor implementation` vs `joiner.go: declares the Joiner implementation type`.

A single source file per *implementation type*,  i.e `range_processor.go` holds an *implementation type* of `Processor` and its *implementation* `NewRange`, its subimplementations `NewRangeBegin` ,`NewRangeEnd`. Exceptions are allowed when they're totally necessary.

A new source file which holds an *implementation type* should be named with the prefix of the noun, the raw description of its *implementation*, i.e `NewMatcher` is inside the `expression_validator.go`,  `NewChain` is inside the `chain_processor.go`,  `NewPrepender` is inside the `prefix_processor.go`. Exceptions are allowed when we are sure that we will contain only one implementation type and only one implementation(and or with one or more functions returning the same implementation with their difference is the passing arguments to the main implementation), even if we already suffix it with an *implementation type*, which is adjective by itself, i.e `replacer_processor.go`, `joiner_processor`, `replacer_processor.go`.


### Functions and Variables

(a) *prefix implementation*:
     the letters-as-word after the `New` and before the `implementation description`.


A basic rule, with its own exceptions, is that any *implementation* **that takes input arguments, aka receivers or parameters, should be named as ADJECTIVE and usually is being prefixed with `New`**, i.e: `NewMatcher`, `NewReplacer`, `NewPrepender`, `NewAppender`, `NewSuffixRemover`.

A `Validator` is possible to be named as a simple noun(if makes sense to the end-user). Can be also prefixed with an `Is` because it's probably to be used as an input argument on `unis.If`, i.e `IsMail`. 

`unis.If(unis.IsMail,...)` is more easier to remember and used than `unis.If(unis.NewMailValidator(), ...)`.


A general and very used as input argument to other implementations, can be declared as a variable.
A variable can omit the `New` keyword if they don't take any arguments for initialization and preparation for its implementation, i.e `ClearProcessor`, `OriginProcessor`, `IsMail`.

**Exceptions are allowed when an adjective as a name doesn't makes sense or it's hard for the end-user to think its name when searching for an implementation**, i.e(`NewChain`, `NewRangeBegin`, `If`)


#### Standard Go Functions

Every *implementation* that just returns a go's  standard package (i.e: `path`, `strings`)
should be *prefixed* with `Native`. 

Take for example a replacer processor which returns a new `(*strings.Replacer).Replace` function,
normally we would name the function as `NewReplacer` but instead we will use the name of `NewNativeReplacer` because it just exposes a native, `strings` standard function without any additions or a different preparation than user could expect from a native function.

For example:

```go
func NewNativeReplacer(oldnew ...string) ProcessorFunc {
	return strings.NewReplacer(oldnew...).Replace
}
```

There are some cases that a `Processor` (or `Divider`, `Joiner`, `Validator`)
prepares the use or return of a standard function, in that case we **don't** prefix them with the `Native` word.

Let's take a similar, to previous, usage. We have a Replacer which makes some preparation and call the `strings.Replace` function
inside itself: 

```go
func NewReplacer(replacements map[string]string) ProcessorFunc {
	replacementsClone := make(map[string]string, len(replacements))

	for old, new := range replacements {
		replacementsClone[old] = new
	}

	return func(original string) (result string) {
		result = original
		for old, new := range replacementsClone {
			result = strings.Replace(result, old, new, -1)
		}

		return
	}
}
```

This function makes preparation, it uses replacements as map, it doesn't use the `strings.Replacer` struct
but it uses the `strings.Replace` (because it's fast), in this case we don't prefix the implementation with the `Native` word.



## Testing

Every new implementation should contain its test, even if it's small or it doesn't cover everything. It should be there in case of any improvements later on.

Follow the test paradigm of the existing testing code on `*_test.go` files, we have a function which can help on the tests.