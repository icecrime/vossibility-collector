package main

// Context is provided to transformations as a way to pass additional data to
// existing templates.
type Context struct {
	// Repository gives information about the repository concerned by the event
	// under transformation.
	Repository RepositoryInfo
}

// RepositoryInfo provides information about the repository to the executed
// templates.
type RepositoryInfo interface {
	// FullName returns the GitHub repository full name, whic is in the form
	// "user/repo" (e.g., "icecrime/docker")
	FullName() string

	// PrettyName returns a vossibility specific string which identifies the
	// repository but also includes its given name (which has no existence on
	// GitHub).
	PrettyName() string
}
