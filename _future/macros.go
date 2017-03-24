package router

/*
TODO:

Here I should think a way to link the framework and user-defined macros
with their one-by-one(?) custom function(s) and all these with one or more PathTmpls or visa-versa

These should be linked at .Boot time, so before the server starts.
The work I have done so far it should be resulted in a single middleware
which will be prepended to the zero position, so no performance cost when no new features are used.
The performance should be the same as now if the path doesn't contains
any macros:
macro    = /api/users/{id:int} or /api/users/{id:int range(1,100) !404}
no macro = /api/users/id

I should add good detailed examples on how a user can override or add his/her
own macros and any optional function(s) can be followed (func = range(1,5)).

Of course no breaking-changes to the user's workflow.
I should not and not need to touch the existing router adaptors.

*/

/* 23 March 2017
Essentially almost finish, on "dirty" code:
- we can define custom macros, custom validators per custom macro or to all of the available macros,
- parse the macro and macro funcs to iris.HandlerFunc and passed as middleware
- use reflection to add custom function signature without need to convert and check for arguments length
(same performance as with the 'hard-manual' way because the func which does all these checks is executed on boot time)

Todo:
- Make this new syntax compatible with the already router adaptors (can be done easy)
- We need to clean the code and think a way to adapt that to Iris in order to be
easy-to-use while in the same time provide the necessary api for advanced use cases.


I should also think if this feature worths the time to be extended into
a complete interpreter(with tokens, lexers, parsers and AST) and name it 'iel/Iris Expression Language' which could be used everywhere in the framework.
*/
