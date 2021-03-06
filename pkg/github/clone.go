package github

import (
	// stdlib

	"fmt"
	"os"
	"sync"

	// external
	"github.com/golang/glog"
	"github.com/google/go-github/github"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"

	// internal
	"github.com/mccurdyc/neighbor/pkg/neighbor"
)

// ClonedRepositoriesCtxKey is used to set a context key as it complains when you use a built-in
// type. This is what is suggested by bradfitz on GitHub https://github.com/golang/go/issues/17826#issuecomment-258946985.
// Error Message: "should not use basic type string as key in context.WithValue"
type ClonedRepositoriesCtxKey struct{}

// repoDirMap will store repository names as the key where the value will be the
// path to where the repository was cloned.
type repoDirMap map[string]string

// ExternalProject contains a GitHub project's name as where it was cloned to
type ExternalProject struct {
	Name      string
	Directory string
}

// CloneFromResult creates temporary directories where the base path is that of os.TempDir
// and the rest of the path is the Name of the repository. After creating a temporary
// directory, a project is cloned into that directory. After creating temp directories
// and cloning projects into the respective directory, the context is updated
// with the project names and the temporary directories.
func CloneFromResult(ctx *neighbor.Ctx, d interface{}) <-chan ExternalProject {
	ch := make(chan ExternalProject) // an unbuffered, synchronous channel for guaranteed delivery

	var wg sync.WaitGroup

	switch t := d.(type) {
	case *github.RepositoriesSearchResult:
		wg.Add(len(t.Repositories))

		for _, r := range t.Repositories {
			go func(repo github.Repository) {
				select {
				case <-ctx.Context.Done():
					wg.Done()
					return
				default:
					cloneRepo(ctx, repo, ch)
					wg.Done()
				}
			}(r)
		}
	case *github.CodeSearchResult:
		wg.Add(len(t.CodeResults))

		for _, r := range t.CodeResults {
			go func(repo github.Repository) {
				select {
				case <-ctx.Context.Done():
					wg.Done()
					return
				default:
					cloneRepo(ctx, repo, ch)
					wg.Done()
				}
			}(*r.Repository)
		}
	}

	go func() {
		wg.Wait()
		// after we are finished cloning the repos and sending them through the pipeline,
		// send a signal informing the consumers that we are done sending.
		close(ch)
	}()

	return ch
}

// getCloneURL returns a GitHub git clone URL e.g., https://github.com/mccurdyc/neighbor.git
func getCloneURL(repo github.Repository) string {
	url := repo.GetCloneURL()
	if url == "" {
		url = fmt.Sprintf("%s.git", repo.GetHTMLURL())
	}
	return url
}

// cloneRepo clones a repository using a GitHub personal access token, given a
// github.Repository to an out directory specified by ctx.ExternalResultDir/repository_name
// and informs downstream consumers of the project name and where it is located
// on the machine.
func cloneRepo(ctx *neighbor.Ctx, repo github.Repository, ch chan<- ExternalProject) {
	glog.V(3).Infof("%+v", repo)

	dir := fmt.Sprintf("%s/%s", ctx.ExtResultDir, *repo.Name)
	glog.V(2).Infof("created directory: %s", dir)

	opts := &git.CloneOptions{
		URL:      getCloneURL(repo),
		Progress: os.Stderr, // Stderr so that it can be surpressed without interfering with external command output
	}

	if len(ctx.GitHub.AccessToken) > 0 {
		// you must use BasicAuth with your GitHub Access Token as the password
		// and the Username can be anything.
		opts.Auth = &http.BasicAuth{
			Username: "abc123", // yes, this can be anything except an empty string
			Password: ctx.GitHub.AccessToken,
		}
	}

	_, err := git.PlainClone(dir, false, opts)
	if err != nil {
		glog.Errorf("failed to clone project %s with error: %+v", *repo.Name, err)
		return
	}

	glog.V(2).Infof("cloned: %s", repo.GetCloneURL())

	// this should block until there is a receiver
	select {
	case <-ctx.Context.Done():
		return
	default:
		ch <- ExternalProject{
			Name:      *repo.Name,
			Directory: dir,
		}
	}
}
