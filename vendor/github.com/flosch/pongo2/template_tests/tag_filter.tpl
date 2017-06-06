{% filter lower %}This is a nice test; let's see whether it works. Foobar. {{ simple.xss }}{% endfilter %}

{% filter truncatechars:10|lower|length %}This is a nice test; let's see whether it works. Foobar. {{ simple.number }}{% endfilter %}