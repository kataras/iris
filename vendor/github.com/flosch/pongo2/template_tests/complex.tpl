{# A more complex template using pongo2 (fully django-compatible template) #}
<!DOCTYPE html>
<html>

<head>
	<title>My blog page</title>
</head>

<body>
	<h1>Blogpost</h1>
	<div id="content">
		{{ complex.post.Text|safe }}
	</div>

	<h1>Comments</h1>

	{% for comment in complex.comments %}
		<h2>{{ forloop.Counter }}. Comment ({{ forloop.Revcounter}} comment{{ forloop.Revcounter|pluralize:"s" }} left)</h2>
		<p>From: {{ comment.Author.Name }} ({{ comment.Author.Validated|yesno:"validated,not validated,unknown validation status" }})</p>

		{% if complex.is_admin(comment.Author) %}
			<p>This user is an admin (verify: {{ comment.Author.Is_admin }})!</p>
		{% else %}
			<p>This user is not admin!</p>
		{% endif %}

		<p>Written {{ comment.Date }}</p>
		<p>{{ comment.Text|striptags }}</p>
	{% endfor %}
</body>

</html>