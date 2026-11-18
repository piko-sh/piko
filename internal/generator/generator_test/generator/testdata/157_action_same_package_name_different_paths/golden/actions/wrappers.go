package actions

import (
	"context"
	repo_495a6466 "testcase_157_action_same_package_name_different_paths/actions/github/repo"
	repo_1bc86c38 "testcase_157_action_same_package_name_different_paths/actions/gitlab/repo"

	pikobinder "piko.sh/piko/wdk/binder"
)

func invokeRepoGithubGet(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	var input repo_495a6466.GithubGetInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		return nil, err
	}
	a := action.(*repo_495a6466.GithubGetAction)
	return a.Call(input)
}
func invokeRepoGitlabGet(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	var input repo_1bc86c38.GitlabGetInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		return nil, err
	}
	a := action.(*repo_1bc86c38.GitlabGetAction)
	return a.Call(input)
}
