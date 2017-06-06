{% for comment in complex.comments %}[{{ forloop.Counter }} {{ forloop.Counter0 }} {{ forloop.First }} {{ forloop.Last }} {{ forloop.Revcounter }} {{ forloop.Revcounter0 }}] {{ comment.Author.Name }}

{# nested loop #}
{% for char in comment.Text %}{{forloop.Parentloop.Counter0}}.{{forloop.Counter0}}:{{ char|safe }} {% endfor %}

{% endfor %}

reversed
'{% for item in simple.multiple_item_list reversed %}{{ item }} {% endfor %}'

sorted string map
'{% for key in simple.strmap sorted %}{{ key }} {% endfor %}'

sorted int map
'{% for key in simple.intmap sorted %}{{ key }} {% endfor %}'

sorted int list
'{% for key in simple.unsorted_int_list sorted %}{{ key }} {% endfor %}'

reversed sorted int list
'{% for key in simple.unsorted_int_list reversed sorted %}{{ key }} {% endfor %}'

reversed sorted string map
'{% for key in simple.strmap reversed sorted %}{{ key }} {% endfor %}'

reversed sorted int map
'{% for key in simple.intmap reversed sorted %}{{ key }} {% endfor %}'
