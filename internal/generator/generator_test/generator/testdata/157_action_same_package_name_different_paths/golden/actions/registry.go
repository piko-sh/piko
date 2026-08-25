package actions

import (
	"context"
	"reflect"
	repo_495a6466 "testcase_157_action_same_package_name_different_paths/actions/github/repo"
	repo_1bc86c38 "testcase_157_action_same_package_name_different_paths/actions/gitlab/repo"

	"piko.sh/piko"
	pikojson "piko.sh/piko/wdk/json"
)

func init() {
	piko.RegisterActions(map[string]piko.ActionHandlerEntry{"repo.GithubGet": {Name: "repo.GithubGet", Method: "POST", Create: func() any {
		return &repo_495a6466.GithubGetAction{}
	}, Invoke: invokeRepoGithubGet, HasSSE: false}, "repo.GitlabGet": {Name: "repo.GitlabGet", Method: "POST", Create: func() any {
		return &repo_1bc86c38.GitlabGetAction{}
	}, Invoke: invokeRepoGitlabGet, HasSSE: false}})
	pretouchTypes := []reflect.Type{reflect.TypeFor[repo_495a6466.GithubGetInput](), reflect.TypeFor[repo_495a6466.GithubGetOutput](), reflect.TypeFor[repo_1bc86c38.GitlabGetInput](), reflect.TypeFor[repo_1bc86c38.GitlabGetOutput]()}
	for _, t := range pretouchTypes {
		_ = pikojson.Pretouch(t)
	}
}

type ActionHandler struct {
	Name   string
	Method string
	Create func() any
	Invoke func(ctx context.Context, action any, args map[string]any) (any, error)
	Bind   func(ctx context.Context, action any, args map[string]any) error
	HasSSE bool
}

func Registry() map[string]ActionHandler {
	return map[string]ActionHandler{"repo.GithubGet": {Name: "repo.GithubGet", Method: "POST", Create: func() any {
		return &repo_495a6466.GithubGetAction{}
	}, Invoke: invokeRepoGithubGet, HasSSE: false}, "repo.GitlabGet": {Name: "repo.GitlabGet", Method: "POST", Create: func() any {
		return &repo_1bc86c38.GitlabGetAction{}
	}, Invoke: invokeRepoGitlabGet, HasSSE: false}}
}
