module myapp

go 1.15

require (
	github.com/google/uuid v1.1.2
	github.com/kataras/iris/v12 v12.2.0-alpha.0.20201106220849-7a19cfb2112f
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897
)

replace github.com/kataras/iris/v12 => ../../../../
