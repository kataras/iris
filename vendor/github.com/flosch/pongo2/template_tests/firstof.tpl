{% firstof doesnotexist 42 %}
{% firstof doesnotexist "<script>alert('xss');</script>" %}
{% firstof doesnotexist "<script>alert('xss');</script>"|safe %}
{% firstof doesnotexist simple.uint 42 %}
{% firstof doesnotexist "test" simple.number 42 %}
{% firstof %}
{% firstof "test" "test2" %}