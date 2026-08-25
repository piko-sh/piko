package actions

import (
	"context"
	repo_495a6466 "testcase_157_action_same_package_name_different_paths/actions/github/repo"
	repo_1bc86c38 "testcase_157_action_same_package_name_different_paths/actions/gitlab/repo"

	pikobinder "piko.sh/piko/wdk/binder"
	"piko.sh/piko/wdk/logger"
)

var pikoActionLog = logger.GetLogger("piko/actions")

func invokeRepoGithubGet(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	ctx, l := logger.From(ctx, pikoActionLog)
	var input repo_495a6466.GithubGetInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		l.Warn("Failed to bind action parameter", logger.String("param", "input"), logger.Error(err))
		return nil, err
	}
	a := action.(*repo_495a6466.GithubGetAction)
	return a.Call(input)
}
func invokeRepoGitlabGet(ctx context.Context, action any, argsMap map[string]any) (any, error) {
	ctx, l := logger.From(ctx, pikoActionLog)
	var input repo_1bc86c38.GitlabGetInput
	if err := pikobinder.BindMap(ctx, &input, pikobinder.ActionInputSource(argsMap, "input"), pikobinder.IgnoreUnknownKeys(true), pikobinder.WithDocumentScaleLimits(), pikobinder.WithValidation(true)); err != nil {
		l.Warn("Failed to bind action parameter", logger.String("param", "input"), logger.Error(err))
		return nil, err
	}
	a := action.(*repo_1bc86c38.GitlabGetAction)
	return a.Call(input)
}
