## Middleware information

This folder contains a middleware ported to Iris for a third-party package named pongo2.

More can be found here:
[https://github.com/flosch/pongo2](https://github.com/flosch/pongo2)


## Description

pongo2 is the successor of pongo, a Django-syntax like templating-language.


## Behaviour


   1. Entirely rewritten from the ground-up.
   2. Advanced C-like expressions.
   3. Complex function calls within expressions.
   4. Easy API to create new filters and tags (including parsing arguments)
   5. Additional features:
        	1. Macros including importing macros from other files (see template_tests/macro.tpl)
        	2. Template sandboxing (directory patterns, banned tags/filters)


## Notes

```HTML+Django

<html><head><title>Our admins and users</title></head>
{# This is a short example to give you a quick overview of pongo2's syntax. #}

{% macro user_details(user, is_admin=false) %}
    <div class="user_item">
        <!-- Let's indicate a user's good karma -->
        <h2 {% if (user.karma >= 40) || (user.karma > calc_avg_karma(userlist)+5) %}
            class="karma-good"{% endif %}>

            <!-- This will call user.String() automatically if available: -->
            {{ user }}
        </h2>

        <!-- Will print a human-readable time duration like "3 weeks ago" -->
        <p>This user registered {{ user.register_date|naturaltime }}.</p>

        <!-- Let's allow the users to write down their biography using markdown;
             we will only show the first 15 words as a preview -->
        <p>The user's biography:</p>
        <p>{{ user.biography|markdown|truncatewords_html:15 }}
            <a href="/user/{{ user.id }}/">read more</a></p>

        {% if is_admin %}<p>This user is an admin!</p>{% endif %}
    </div>
{% endmacro %}

<body>
    <!-- Make use of the macro defined above to avoid repetitive HTML code
         since we want to use the same code for admins AND members -->

    <h1>Our admins</h1>
    {% for admin in adminlist %}
        {{ user_details(admin, true) }}
    {% endfor %}

    <h1>Our members</h1>
    {% for user in userlist %}
        {{ user_details(user) }}
    {% endfor %}
</body>
</html>

```
## Usage

```go

package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware/pongo2"
)

func main() {
	iris.Use(pongo2.New())

	iris.Get("/", func(ctx *iris.Context) {
		ctx.Set("template", "index.html")
		ctx.Set("data", map[string]interface{}{"message": "Hello World!"})
	})

	println("Server is running at :8080")
	iris.Listen(":8080")
}


```


## Using Static files with pongo2


```go
//...
iris.Static("/img", "./public/img", 1)
iris.Static("/js", "./public/js", 1)
iris.Static("/css", "./public/css", 1)

iris.Use(pongo2.Pongo2()) //after this
//...
```

and on the template file add a style this way:

```html
<link rel="stylesheet" type="text/css" href="./css/yourstyle.css">
```
