{% for item in simple.multiple_item_list %}
    '{% cycle "item1" simple.name simple.number %}'
{% endfor %}
{% for item in simple.multiple_item_list %}
    '{% cycle "item1" simple.name simple.number as cycleitem %}'
    May I present the cycle item again: '{{ cycleitem }}'
{% endfor %}
{% for item in simple.multiple_item_list %}
    '{% cycle "item1" simple.name simple.number as cycleitem silent %}'
    May I present the cycle item: '{{ cycleitem }}'
{% endfor %}
{% for item in simple.multiple_item_list %}
    '{% cycle "item1" simple.name simple.number as cycleitem silent %}'
    May I present the cycle item: '{{ cycleitem }}'
    {% include "inheritance/cycle_include.tpl" %}
{% endfor %}
'{% cycle "item1" simple.name simple.number as cycleitem %}'
'{% cycle cycleitem %}'
'{% cycle "item1" simple.name simple.number as cycleitem silent %}'
'{{ cycleitem }}'
'{% cycle cycleitem %}'
'{{ cycleitem }}'