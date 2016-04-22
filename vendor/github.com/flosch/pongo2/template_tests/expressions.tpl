integers and complex expressions
{{ 10-100 }}
{{ -(10-100) }}
{{ -(-(10-100)) }}
{{ -1 * (-(-(10-100))) }}
{{ -1 * (-(-(10-100)) ^ 2) ^ 3 + 3 * (5 - 17) + 1 + 2 }}

floats
{{ 5.5 }}
{{ 5.172841 }}
{{ 5.5 - 1.5 == 4 }}
{{ 5.5 - 1.5 == 4.0 }}

mul/div
{{ 2 * 5 }}
{{ 2 * 5.0 }}
{{ 2 * 0 }}
{{ 2.5 * 5.3 }}
{{ 1/2 }}
{{ 1/2.0 }}
{{ 1/0.000001 }}

logic expressions
{{ !true }}
{{ !(true || false) }}
{{ true || false }}
{{ true or false }}
{{ false or false }}
{{ false || false }}
{{ true && (true && (true && (true && (1 == 1 || false)))) }}

float comparison
{{ 5.5 <= 5.5 }}
{{ 5.5 < 5.5 }}
{{ 5.5 > 5.5 }}
{{ 5.5 >= 5.5 }}

remainders
{{ (simple.number+7)%7 }}
{{ (simple.number+7)%7 == 0 }}
{{ (simple.number+7)%6 }}

in/not in
{{ 5 in simple.intmap }}
{{ 2 in simple.intmap }}
{{ 7 in simple.intmap }}
{{ !(5 in simple.intmap) }}
{{ not(7 in simple.intmap) }}
{{ 1 in simple.multiple_item_list }}
{{ 4 in simple.multiple_item_list }}
{{ !(4 in simple.multiple_item_list) }}
{{ "Hello" in simple.misc_list }}
{{ "Hello2" in simple.misc_list }}
{{ 99 in simple.misc_list }}
{{ False in simple.misc_list }}

issue #48 (associativity for infix operators)
{{ 34/3*3 }}
{{ 10 + 24 / 6 / 2 }}
{{ 6 - 4 - 2 }}

issue #64 (uint comparison with int const)
{{ simple.uint }}
{{ simple.uint == 8 }}
{{ simple.uint == 9 }}
{{ simple.uint >= 8 }}
{{ simple.uint <= 8 }}
{{ simple.uint < 8 }}
{{ simple.uint > 8 }}