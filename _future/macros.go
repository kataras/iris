package router

/*
TODO:

Here I should think a way to link the framework and user-defined macros
with their one-by-one(?) custom function(s) and all these with one or more PathTmpls or visa-versa

These should be linked at .Boot time, so before the server starts.
Tthe work I have done so far it should be resulted in a single middleware
which will be prepended to the zero position, so no performance cost when no new features are used.
The performance should be the same as now if the path doesn't contains
any macros:
macro    = /api/users/{id:int} or /api/users/{id:int range(1,100) !404}
no macro = /api/users/id}).

I should add a good detailed examples on how the user can override or add his/her
own macros and optional functions can be followed (i.e, func = range(1,5)).

Of course no breaking-changes to the user's workflow(I should not and not need to touch the existing router adaptors).

*/
