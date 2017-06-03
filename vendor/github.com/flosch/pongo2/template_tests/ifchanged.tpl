{% for comment in complex.comments2 %}
    {% ifchanged %}New comment from another user {{ comment.Author.Name }}{% endifchanged %}
    {% ifchanged comment.Author.Validated %}
        Validated changed to {{ comment.Author.Validated }}
    {% else %}
        Validated value not changed
    {% endifchanged %}
    {% ifchanged comment.Author.Name comment.Date %}Comment's author name or date changed{% endifchanged %}
{% endfor %}