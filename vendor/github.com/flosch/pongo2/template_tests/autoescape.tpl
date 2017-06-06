{{ "<script>alert('xss');</script>" }}
{% autoescape off %}
{{ "<script>alert('xss');</script>" }}
{% endautoescape %}
{% autoescape on %}
{{ "<script>alert('xss');</script>" }}
{% endautoescape %}
{% autoescape off %}
{{ "<script>alert('xss');</script>"|escape }}
{% endautoescape %}