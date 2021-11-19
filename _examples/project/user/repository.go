package user

type Repository interface { // Repo methods here...
}

type repo struct { // Hold database instance here: e.g.
	// *mydatabase_pkg.DB
}

func NewRepository( /*  *mydatabase_pkg.DB */ ) Repository {
	return &repo{ /* db: db */ }
}
