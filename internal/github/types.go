package github

type Repository struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Owner    User   `json:"owner"`
	Private  bool   `json:"private"`
}

type User struct {
	Login string `json:"login"`
}

type PullRequest struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	State     string `json:"state"`
	HTMLURL   string `json:"html_url"`
	Body      string `json:"body"`
	Mergeable *bool  `json:"mergeable"`
}

type Branch struct {
	Name string `json:"name"`
}
type PullRequestInput struct{ Title, Body, Head, Base string }
type DiffFile struct {
	Filename, Status, Patch, HTMLURL string
	Additions, Deletions             int
}

type Workflow struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Path  string `json:"path"`
	State string `json:"state"`
}

type WorkflowRun struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	HTMLURL    string `json:"html_url"`
}

type Job struct {
	ID                       int64 `json:"id"`
	Name, Status, Conclusion string
	Steps                    []Step `json:"steps"`
}
type Step struct {
	Name, Status, Conclusion string
	Number                   int `json:"number"`
}
