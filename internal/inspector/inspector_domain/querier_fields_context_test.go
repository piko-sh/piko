// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package inspector_domain

import (
	"context"
	goast "go/ast"
	"testing"

	"github.com/stretchr/testify/mock"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

type mockFieldFinder struct {
	mock.Mock
}

func (m *mockFieldFinder) findFieldInfoSingleSegment(
	baseType goast.Expr,
	fieldName string,
	importerPackagePath, importerFilePath string,
) *inspector_dto.FieldInfo {

	arguments := m.Called(mock.Anything, fieldName, importerPackagePath, importerFilePath)

	if arguments.Get(0) == nil {
		return nil
	}
	result, ok := arguments.Get(0).(*inspector_dto.FieldInfo)
	if !ok {
		return nil
	}
	return result
}

func (m *mockFieldFinder) updateContextForNextSegment(info *inspector_dto.FieldInfo) (nextPackage, nextFile string) {
	arguments := m.Called(info)
	return arguments.String(0), arguments.String(1)
}

func TestFindFieldInfoDeep_ContextSwitching(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		setupMocks func(m *mockFieldFinder)
		name       string
		fieldPath  string
	}{

		{
			name:      "Deep field access with package alias context switch",
			fieldPath: "ServiceResponse.Transaction.Amount",
			setupMocks: func(m *mockFieldFinder) {

				m.On("findFieldInfoSingleSegment", mock.Anything, "ServiceResponse", "main_pkg", "main.go").
					Return(&inspector_dto.FieldInfo{
						Type:                 goastutil.TypeStringToAST("services.TransactionServiceResponse"),
						CanonicalPackagePath: "services_pkg", DefiningFilePath: "services/transaction.go",
					}).Once()
				m.On("updateContextForNextSegment", mock.Anything).Return("services_pkg", "services/transaction.go").Once()

				m.On("findFieldInfoSingleSegment", mock.Anything, "Transaction", "services_pkg", "services/transaction.go").
					Return(&inspector_dto.FieldInfo{
						Type:                 goastutil.TypeStringToAST("dto_alias.TransactionDto"),
						CanonicalPackagePath: "dtos_pkg", DefiningFilePath: "dtos/transaction.go",
					}).Once()
				m.On("updateContextForNextSegment", mock.Anything).Return("dtos_pkg", "dtos/transaction.go").Once()

				m.On("findFieldInfoSingleSegment", mock.Anything, "Amount", "dtos_pkg", "dtos/transaction.go").
					Return(&inspector_dto.FieldInfo{
						Type:                 goastutil.TypeStringToAST("maths.Money"),
						CanonicalPackagePath: "maths_pkg", DefiningFilePath: "maths/money.go",
					}).Once()
			},
		},

		{
			name:      "Simple deep access within the same package",
			fieldPath: "User.Profile.Email",
			setupMocks: func(m *mockFieldFinder) {
				m.On("findFieldInfoSingleSegment", mock.Anything, "User", "main_pkg", "main.go").
					Return(&inspector_dto.FieldInfo{
						Type: goastutil.TypeStringToAST("main.User"), CanonicalPackagePath: "main_pkg", DefiningFilePath: "main.go",
					}).Once()
				m.On("updateContextForNextSegment", mock.Anything).Return("main_pkg", "main.go").Once()

				m.On("findFieldInfoSingleSegment", mock.Anything, "Profile", "main_pkg", "main.go").
					Return(&inspector_dto.FieldInfo{
						Type: goastutil.TypeStringToAST("main.Profile"), CanonicalPackagePath: "main_pkg", DefiningFilePath: "main.go",
					}).Once()
				m.On("updateContextForNextSegment", mock.Anything).Return("main_pkg", "main.go").Once()

				m.On("findFieldInfoSingleSegment", mock.Anything, "Email", "main_pkg", "main.go").
					Return(&inspector_dto.FieldInfo{
						Type: goastutil.TypeStringToAST("string"),
					}).Once()
			},
		},

		{
			name:      "Fails on the second segment of the path",
			fieldPath: "Response.InvalidField.ID",
			setupMocks: func(m *mockFieldFinder) {
				m.On("findFieldInfoSingleSegment", mock.Anything, "Response", "main_pkg", "main.go").
					Return(&inspector_dto.FieldInfo{
						Type: goastutil.TypeStringToAST("api.Response"), CanonicalPackagePath: "api_pkg", DefiningFilePath: "api/response.go",
					}).Once()
				m.On("updateContextForNextSegment", mock.Anything).Return("api_pkg", "api/response.go").Once()

				m.On("findFieldInfoSingleSegment", mock.Anything, "InvalidField", "api_pkg", "api/response.go").
					Return(nil).Once()

			},
		},

		{
			name:      "Single segment path does not trigger context update",
			fieldPath: "User",
			setupMocks: func(m *mockFieldFinder) {
				m.On("findFieldInfoSingleSegment", mock.Anything, "User", "main_pkg", "main.go").
					Return(&inspector_dto.FieldInfo{
						Type: goastutil.TypeStringToAST("main.User"),
					}).Once()

			},
		},

		{
			name:      "Path contains a pointer type",
			fieldPath: "Manager.Name",
			setupMocks: func(m *mockFieldFinder) {
				m.On("findFieldInfoSingleSegment", mock.Anything, "Manager", "main_pkg", "main.go").
					Return(&inspector_dto.FieldInfo{
						Type: goastutil.TypeStringToAST("*models.User"), CanonicalPackagePath: "models_pkg", DefiningFilePath: "models/user.go",
					}).Once()
				m.On("updateContextForNextSegment", mock.Anything).Return("models_pkg", "models/user.go").Once()

				m.On("findFieldInfoSingleSegment", mock.Anything, "Name", "models_pkg", "models/user.go").
					Return(&inspector_dto.FieldInfo{
						Type: goastutil.TypeStringToAST("string"),
					}).Once()
			},
		},

		{
			name:      "Path contains a slice type",
			fieldPath: "Users.Name",
			setupMocks: func(m *mockFieldFinder) {
				m.On("findFieldInfoSingleSegment", mock.Anything, "Users", "main_pkg", "main.go").
					Return(&inspector_dto.FieldInfo{
						Type: goastutil.TypeStringToAST("[]models.User"), CanonicalPackagePath: "models_pkg", DefiningFilePath: "models/user.go",
					}).Once()
				m.On("updateContextForNextSegment", mock.Anything).Return("models_pkg", "models/user.go").Once()

				m.On("findFieldInfoSingleSegment", mock.Anything, "Name", "models_pkg", "models/user.go").
					Return(nil).Once()
			},
		},

		{
			name:      "Context switches to a different file in the same package",
			fieldPath: "Config.Settings",
			setupMocks: func(m *mockFieldFinder) {
				m.On("findFieldInfoSingleSegment", mock.Anything, "Config", "main_pkg", "main.go").
					Return(&inspector_dto.FieldInfo{
						Type: goastutil.TypeStringToAST("main.Config"), CanonicalPackagePath: "main_pkg", DefiningFilePath: "config.go",
					}).Once()
				m.On("updateContextForNextSegment", mock.Anything).Return("main_pkg", "config.go").Once()

				m.On("findFieldInfoSingleSegment", mock.Anything, "Settings", "main_pkg", "config.go").
					Return(&inspector_dto.FieldInfo{
						Type: goastutil.TypeStringToAST("main.Settings"),
					}).Once()
			},
		},

		{
			name:      "Field's type is primitive and has no canonical path",
			fieldPath: "Count.NonExistent",
			setupMocks: func(m *mockFieldFinder) {
				m.On("findFieldInfoSingleSegment", mock.Anything, "Count", "main_pkg", "main.go").
					Return(&inspector_dto.FieldInfo{
						Type: goastutil.TypeStringToAST("int"), CanonicalPackagePath: "", DefiningFilePath: "main.go", DefiningPackagePath: "main_pkg",
					}).Once()

				m.On("updateContextForNextSegment", mock.Anything).Return("main_pkg", "main.go").Once()

				m.On("findFieldInfoSingleSegment", mock.Anything, "NonExistent", "main_pkg", "main.go").
					Return(nil).Once()
			},
		},

		{
			name:      "Very long path with multiple context switches",
			fieldPath: "A.B.C.D.E",
			setupMocks: func(m *mockFieldFinder) {

				m.On("findFieldInfoSingleSegment", mock.Anything, "A", "main_pkg", "main.go").
					Return(&inspector_dto.FieldInfo{
						Name:                 "A",
						Type:                 goastutil.TypeStringToAST("pkgB.TypeB"),
						CanonicalPackagePath: "pkgB", DefiningFilePath: "b.go",
					}).Once()

				m.On("updateContextForNextSegment", mock.Anything).Return("pkgB", "b.go").Once()

				m.On("findFieldInfoSingleSegment", mock.Anything, "B", "pkgB", "b.go").
					Return(&inspector_dto.FieldInfo{
						Name:                 "B",
						Type:                 goastutil.TypeStringToAST("pkgC.TypeC"),
						CanonicalPackagePath: "pkgC", DefiningFilePath: "c.go",
					}).Once()
				m.On("updateContextForNextSegment", mock.Anything).Return("pkgC", "c.go").Once()

				m.On("findFieldInfoSingleSegment", mock.Anything, "C", "pkgC", "c.go").
					Return(&inspector_dto.FieldInfo{
						Name:                 "C",
						Type:                 goastutil.TypeStringToAST("pkgD.TypeD"),
						CanonicalPackagePath: "pkgD", DefiningFilePath: "d.go",
					}).Once()
				m.On("updateContextForNextSegment", mock.Anything).Return("pkgD", "d.go").Once()

				m.On("findFieldInfoSingleSegment", mock.Anything, "D", "pkgD", "d.go").
					Return(&inspector_dto.FieldInfo{
						Name:                 "D",
						Type:                 goastutil.TypeStringToAST("pkgE.TypeE"),
						CanonicalPackagePath: "pkgE", DefiningFilePath: "e.go",
					}).Once()
				m.On("updateContextForNextSegment", mock.Anything).Return("pkgE", "e.go").Once()

				m.On("findFieldInfoSingleSegment", mock.Anything, "E", "pkgE", "e.go").
					Return(&inspector_dto.FieldInfo{
						Name: "E",
						Type: goastutil.TypeStringToAST("string"),
					}).Once()
			},
		},

		{
			name:      "Intermediate field is an alias to a type in another package",
			fieldPath: "Wrapper.AliasedItem.Name",
			setupMocks: func(m *mockFieldFinder) {
				m.On("findFieldInfoSingleSegment", mock.Anything, "Wrapper", "main_pkg", "main.go").
					Return(&inspector_dto.FieldInfo{
						Type: goastutil.TypeStringToAST("models.Wrapper"), CanonicalPackagePath: "models_pkg", DefiningFilePath: "models/wrapper.go",
					}).Once()
				m.On("updateContextForNextSegment", mock.Anything).Return("models_pkg", "models/wrapper.go").Once()

				m.On("findFieldInfoSingleSegment", mock.Anything, "AliasedItem", "models_pkg", "models/wrapper.go").
					Return(&inspector_dto.FieldInfo{
						Type: goastutil.TypeStringToAST("other.Item"), CanonicalPackagePath: "other_pkg", DefiningFilePath: "other/item.go",
					}).Once()
				m.On("updateContextForNextSegment", mock.Anything).Return("other_pkg", "other/item.go").Once()

				m.On("findFieldInfoSingleSegment", mock.Anything, "Name", "other_pkg", "other/item.go").
					Return(&inspector_dto.FieldInfo{
						Type: goastutil.TypeStringToAST("string"),
					}).Once()
			},
		},

		{
			name:      "Final segment of a valid path fails",
			fieldPath: "User.Profile.NonExistentField",
			setupMocks: func(m *mockFieldFinder) {
				m.On("findFieldInfoSingleSegment", mock.Anything, "User", "main_pkg", "main.go").
					Return(&inspector_dto.FieldInfo{Type: goastutil.TypeStringToAST("main.User"), CanonicalPackagePath: "main_pkg", DefiningFilePath: "main.go"}).Once()
				m.On("updateContextForNextSegment", mock.Anything).Return("main_pkg", "main.go").Once()

				m.On("findFieldInfoSingleSegment", mock.Anything, "Profile", "main_pkg", "main.go").
					Return(&inspector_dto.FieldInfo{Type: goastutil.TypeStringToAST("main.Profile"), CanonicalPackagePath: "main_pkg", DefiningFilePath: "main.go"}).Once()
				m.On("updateContextForNextSegment", mock.Anything).Return("main_pkg", "main.go").Once()

				m.On("findFieldInfoSingleSegment", mock.Anything, "NonExistentField", "main_pkg", "main.go").
					Return(nil).Once()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockFinder := new(mockFieldFinder)
			tc.setupMocks(mockFinder)

			findFieldInfoDeep(
				context.Background(),
				mockFinder,
				goastutil.TypeStringToAST("main.State"),
				tc.fieldPath,
				"main_pkg",
				"main.go",
			)

			mockFinder.AssertExpectations(t)
		})
	}
}
