new style
Start '{% with what_am_i=simple.name %}I'm {{what_am_i}}{% endwith %}' End
Start '{% with what_am_i=simple.name %}I'm {{what_am_i}}11{% endwith %}' End
Start '{% with number=7 what_am_i="guest" %}I'm {{what_am_i}}{{number}}{% endwith %}' End
Start '{% include "with.helper" with what_am_i=simple.name number=10 %}' End

old style - still supported by Django
Start '{% with simple.name as what_am_i %}I'm {{what_am_i}}{% endwith %}' End
Start '{% with simple.name as what_am_i %}I'm {{what_am_i}}11{% endwith %}' End
Start '{% with 7 as number "guest" as what_am_i %}I'm {{what_am_i}}{{number}}{% endwith %}' End
Start '{% include "with.helper" with what_am_i=simple.name number=10 %}' End

more with tests
{% with first_comment=complex.comments|first %}{{ first_comment.Author }}{% endwith %}
{% with first_comment=complex.comments|first %}{{ first_comment.Author.Name }}{% endwith %}
{% with first_comment=complex.comments|last %}{{ first_comment.Author.Name }}{% endwith %}