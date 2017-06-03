empty single line comment
{# #}

filled single line comment
{# testing single line comment #}

filled single line comment with valid tags
{# testing single line comment {% if thing %}{% endif %} #}

filled single line comment with invalid tags
{# testing single line comment {% if thing %} #}

filled single line comment with invalid syntax
{# testing single line comment {% if thing('') %}wow{% endif %} #}

empty block comment
{% comment %}{% endcomment %}

filled text single line block comment
{% comment %}filled block comment {% endcomment %}

empty multi line block comment
{% comment %}


{% endcomment %}

block comment with other tags inside of it
{% comment %}
  {{ thing_goes_here }}
  {% if stuff %}do stuff{% endif %}
{% endcomment %}

block comment with invalid tags inside of it
{% comment %}
  {% if thing %}
{% endcomment %}

block comment with invalid syntax inside of it
{% comment %}
  {% thing('') %}
{% endcomment %}

Regular tags between comments to verify it doesn't break in the lexer
{% if hello %}
{% endif %}
after if
{% comment %}All done{% endcomment %}

end of file
