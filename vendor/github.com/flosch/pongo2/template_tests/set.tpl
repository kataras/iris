{% set new_var = "hello" %}{{ new_var }}
{% block content %}{% set new_var = "world" %}{{ new_var }}{% endblock %}
{{ new_var }}{% for item in simple.misc_list %}
{% set new_var = item %}{{ new_var }}{% endfor %}
{{ new_var }}
{% set car=someUndefinedVar %}{{ car.Drive }}No Panic