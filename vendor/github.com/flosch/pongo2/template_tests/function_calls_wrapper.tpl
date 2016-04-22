{{ simple.func_add(simple.func_add(5, 15), simple.number) + 17 }}
{{ simple.func_add_iface(simple.func_add_iface(5, 15), simple.number) + 17 }}
{{ simple.func_variadic("hello") }}
{{ simple.func_variadic("hello, %s", simple.name) }}
{{ simple.func_variadic("%d + %d %s %d", 5, simple.number, "is", 49) }}
{{ simple.func_variadic_sum_int() }}
{{ simple.func_variadic_sum_int(1) }}
{{ simple.func_variadic_sum_int(1, 19, 185) }}
{{ simple.func_variadic_sum_int2() }}
{{ simple.func_variadic_sum_int2(2) }}
{{ simple.func_variadic_sum_int2(1, 7, 100) }}