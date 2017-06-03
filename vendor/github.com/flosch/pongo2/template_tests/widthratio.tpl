{# Tip: In pongo2 you can easily use arithmetic expressions like value/100.0, but widthratio is supported as well #}
{% widthratio 175 200 100 %}
{% widthratio 175 200 100 as width %}
{{ width }}