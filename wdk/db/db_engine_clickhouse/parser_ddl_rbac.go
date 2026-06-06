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

package db_engine_clickhouse

import (
	"fmt"

	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// keywordUser is the RBAC object keyword for USER statements.
	keywordUser = "USER"

	// keywordRole is the RBAC object keyword for ROLE statements.
	keywordRole = "ROLE"

	// keywordPolicy is the RBAC object keyword for POLICY statements.
	keywordPolicy = "POLICY"

	// keywordRow is the optional ROW qualifier preceding POLICY.
	keywordRow = "ROW"

	// keywordQuota is the RBAC object keyword for QUOTA statements.
	keywordQuota = "QUOTA"

	// keywordSettings is the SETTINGS qualifier preceding PROFILE.
	keywordSettings = "SETTINGS"

	// keywordProfile is the RBAC object keyword for SETTINGS PROFILE statements.
	keywordProfile = "PROFILE"

	// engineKeyUserName is the EngineSpecific key under which a parsed user name is kept.
	engineKeyUserName = "USER_NAME"

	// engineKeyRoleName is the EngineSpecific key under which a parsed role name is kept.
	engineKeyRoleName = "ROLE_NAME"

	// engineKeyPolicyName is the EngineSpecific key under which a parsed policy name is
	// stored.
	engineKeyPolicyName = "POLICY_NAME"

	// engineKeyQuotaName is the EngineSpecific key under which a parsed quota name is
	// stored.
	engineKeyQuotaName = "QUOTA_NAME"

	// engineKeyProfileName is the EngineSpecific key under which a parsed profile name is
	// stored.
	engineKeyProfileName = "PROFILE_NAME"

	// engineKeyRBACKind is the EngineSpecific key recording the GRANT or REVOKE kind.
	engineKeyRBACKind = "RBAC_KIND"

	// errExpectedKeyword is the canonical message format used by the RBAC parsers when a
	// required object keyword is missing.
	errExpectedKeyword = "expected %s at position %d"
)

// parseCreateUser handles `CREATE USER [IF NOT EXISTS | OR REPLACE] name [, ...] [...]`.
//
// Returns *querier_dto.CatalogueMutation which is the parsed CREATE USER mutation.
// Returns error when the statement cannot be parsed.
func (p *parser) parseCreateUser() (*querier_dto.CatalogueMutation, error) {
	return p.parseRBACCreate(keywordUser, querier_dto.MutationCreateUser, engineKeyUserName)
}

// parseCreateRole handles `CREATE ROLE [IF NOT EXISTS | OR REPLACE] name [, ...] [...]`.
//
// Returns *querier_dto.CatalogueMutation which is the parsed CREATE ROLE mutation.
// Returns error when the statement cannot be parsed.
func (p *parser) parseCreateRole() (*querier_dto.CatalogueMutation, error) {
	return p.parseRBACCreate(keywordRole, querier_dto.MutationCreateRole, engineKeyRoleName)
}

// parseCreatePolicy handles `CREATE [ROW] POLICY [IF NOT EXISTS | OR REPLACE] name [,
// ...] [...]`.
//
// Returns *querier_dto.CatalogueMutation which is the parsed CREATE POLICY mutation.
// Returns error when the statement cannot be parsed.
func (p *parser) parseCreatePolicy() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCreate)
	p.skipCreatePrefixesInParser()
	p.matchKeyword(keywordRow)
	return p.captureRBACPolicyOrProfile(keywordPolicy, querier_dto.MutationCreatePolicy, engineKeyPolicyName, true)
}

// parseCreateQuota handles `CREATE QUOTA [IF NOT EXISTS | OR REPLACE] name [, ...]
// [...]`.
//
// Returns *querier_dto.CatalogueMutation which is the parsed CREATE QUOTA mutation.
// Returns error when the statement cannot be parsed.
func (p *parser) parseCreateQuota() (*querier_dto.CatalogueMutation, error) {
	return p.parseRBACCreate(keywordQuota, querier_dto.MutationCreateQuota, engineKeyQuotaName)
}

// parseCreateSettingsProfile handles `CREATE SETTINGS PROFILE [IF NOT EXISTS | OR
// REPLACE] name [, ...] [...]`.
//
// Returns *querier_dto.CatalogueMutation which is the parsed CREATE SETTINGS PROFILE
// mutation.
// Returns error when the statement cannot be parsed.
func (p *parser) parseCreateSettingsProfile() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCreate)
	p.skipCreatePrefixesInParser()
	p.matchKeyword(keywordSettings)
	return p.captureRBACPolicyOrProfile(keywordProfile, querier_dto.MutationCreateSettingsProfile, engineKeyProfileName, true)
}

// captureRBACPolicyOrProfile consolidates the parse-name-and-body step shared by the
// POLICY and PROFILE variants of CREATE, ALTER, and DROP.
//
// When isCreate is true the modifier branch matches matchIfNotExists (CREATE); otherwise
// it matches matchIfExists (ALTER and DROP).
//
// Takes objectKeyword (string) which is the trailing object word (POLICY or PROFILE).
// Takes kind (querier_dto.MutationKind) which is the mutation kind to emit.
// Takes nameKey (string) which is the EngineSpecific key under which to store the name.
// Takes isCreate (bool) which selects the CREATE rather than the ALTER/DROP modifier
// shape.
//
// Returns *querier_dto.CatalogueMutation which is the parsed mutation.
// Returns error when the object keyword is missing or the name cannot be parsed.
func (p *parser) captureRBACPolicyOrProfile(objectKeyword string, kind querier_dto.MutationKind, nameKey string, isCreate bool) (*querier_dto.CatalogueMutation, error) {
	if !p.matchKeyword(objectKeyword) {
		return nil, fmt.Errorf(errExpectedKeyword, objectKeyword, p.current().position)
	}
	if isCreate {
		p.matchRBACCreateModifiers()
	} else {
		p.matchIfExists()
	}
	name, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}
	return &querier_dto.CatalogueMutation{
		Kind: kind,
		EngineSpecific: map[string]string{
			nameKey:                name,
			engineKeyStatementBody: p.consumeRemainderAsText(),
		},
	}, nil
}

// matchRBACCreateModifiers consumes the optional create modifier that ClickHouse RBAC
// statements place after the object keyword, as in `CREATE USER [IF NOT EXISTS | OR
// REPLACE] name`.
//
// The pre-keyword skipCreatePrefixesInParser only covers the `CREATE OR REPLACE TABLE`
// shape used by tables and views; for RBAC the modifier follows the keyword, so without
// this consume `OR REPLACE` would be misread as the object name. Only one of the two
// modifiers is legal, so a single branch suffices.
func (p *parser) matchRBACCreateModifiers() {
	if p.matchKeyword("OR") {
		p.matchKeyword("REPLACE")
		return
	}
	p.matchIfNotExists()
}

// parseRBACCreate is the shared body of the RBAC CREATE handlers (USER, ROLE, QUOTA).
//
// Takes objectKeyword (string) which is the object word to match after CREATE.
// Takes kind (querier_dto.MutationKind) which is the mutation kind to emit.
// Takes nameKey (string) which is the EngineSpecific key under which to store the name.
//
// Returns *querier_dto.CatalogueMutation which is the parsed CREATE mutation.
// Returns error when the object keyword is missing or the name cannot be parsed.
func (p *parser) parseRBACCreate(objectKeyword string, kind querier_dto.MutationKind, nameKey string) (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordCreate)
	p.skipCreatePrefixesInParser()
	if !p.matchKeyword(objectKeyword) {
		return nil, fmt.Errorf(errExpectedKeyword, objectKeyword, p.current().position)
	}
	p.matchRBACCreateModifiers()
	name, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}
	return &querier_dto.CatalogueMutation{
		Kind: kind,
		EngineSpecific: map[string]string{
			nameKey:                name,
			engineKeyStatementBody: p.consumeRemainderAsText(),
		},
	}, nil
}

// parseAlterUser handles `ALTER USER [IF EXISTS] name [, ...] [...]`.
//
// Returns *querier_dto.CatalogueMutation which is the parsed ALTER USER mutation.
// Returns error when the statement cannot be parsed.
func (p *parser) parseAlterUser() (*querier_dto.CatalogueMutation, error) {
	return p.parseRBACAlter(keywordUser, querier_dto.MutationAlterUser, engineKeyUserName)
}

// parseAlterRole handles `ALTER ROLE [IF EXISTS] name [, ...] [...]`.
//
// Returns *querier_dto.CatalogueMutation which is the parsed ALTER ROLE mutation.
// Returns error when the statement cannot be parsed.
func (p *parser) parseAlterRole() (*querier_dto.CatalogueMutation, error) {
	return p.parseRBACAlter(keywordRole, querier_dto.MutationAlterRole, engineKeyRoleName)
}

// parseAlterPolicy handles `ALTER [ROW] POLICY [IF EXISTS] name [, ...] [...]`.
//
// Returns *querier_dto.CatalogueMutation which is the parsed ALTER POLICY mutation.
// Returns error when the statement cannot be parsed.
func (p *parser) parseAlterPolicy() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword("ALTER")
	p.matchKeyword(keywordRow)
	return p.captureRBACPolicyOrProfile(keywordPolicy, querier_dto.MutationAlterPolicy, engineKeyPolicyName, false)
}

// parseAlterQuota handles `ALTER QUOTA [IF EXISTS] name [, ...] [...]`.
//
// Returns *querier_dto.CatalogueMutation which is the parsed ALTER QUOTA mutation.
// Returns error when the statement cannot be parsed.
func (p *parser) parseAlterQuota() (*querier_dto.CatalogueMutation, error) {
	return p.parseRBACAlter(keywordQuota, querier_dto.MutationAlterQuota, engineKeyQuotaName)
}

// parseAlterSettingsProfile handles `ALTER SETTINGS PROFILE [IF EXISTS] name [, ...]
// [...]`.
//
// Returns *querier_dto.CatalogueMutation which is the parsed ALTER SETTINGS PROFILE
// mutation.
// Returns error when the statement cannot be parsed.
func (p *parser) parseAlterSettingsProfile() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword("ALTER")
	p.matchKeyword(keywordSettings)
	return p.captureRBACPolicyOrProfile(keywordProfile, querier_dto.MutationAlterSettingsProfile, engineKeyProfileName, false)
}

// parseRBACAlter is the shared body of the RBAC ALTER handlers (USER, ROLE, QUOTA).
//
// Takes objectKeyword (string) which is the object word to match after ALTER.
// Takes kind (querier_dto.MutationKind) which is the mutation kind to emit.
// Takes nameKey (string) which is the EngineSpecific key under which to store the name.
//
// Returns *querier_dto.CatalogueMutation which is the parsed ALTER mutation.
// Returns error when the object keyword is missing or the name cannot be parsed.
func (p *parser) parseRBACAlter(objectKeyword string, kind querier_dto.MutationKind, nameKey string) (*querier_dto.CatalogueMutation, error) {
	return p.parseRBACAlterOrDrop("ALTER", objectKeyword, kind, nameKey)
}

// parseDropUser handles `DROP USER [IF EXISTS] name [, ...] [ON CLUSTER c]`.
//
// Returns *querier_dto.CatalogueMutation which is the parsed DROP USER mutation.
// Returns error when the statement cannot be parsed.
func (p *parser) parseDropUser() (*querier_dto.CatalogueMutation, error) {
	return p.parseRBACDrop(keywordUser, querier_dto.MutationDropUser, engineKeyUserName)
}

// parseDropRole handles `DROP ROLE [IF EXISTS] name [, ...] [ON CLUSTER c]`.
//
// Returns *querier_dto.CatalogueMutation which is the parsed DROP ROLE mutation.
// Returns error when the statement cannot be parsed.
func (p *parser) parseDropRole() (*querier_dto.CatalogueMutation, error) {
	return p.parseRBACDrop(keywordRole, querier_dto.MutationDropRole, engineKeyRoleName)
}

// parseDropPolicy handles `DROP [ROW] POLICY [IF EXISTS] name [, ...] [...]`.
//
// Returns *querier_dto.CatalogueMutation which is the parsed DROP POLICY mutation.
// Returns error when the statement cannot be parsed.
func (p *parser) parseDropPolicy() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordDrop)
	p.matchKeyword(keywordRow)
	return p.captureRBACPolicyOrProfile(keywordPolicy, querier_dto.MutationDropPolicy, engineKeyPolicyName, false)
}

// parseDropQuota handles `DROP QUOTA [IF EXISTS] name [, ...] [ON CLUSTER c]`.
//
// Returns *querier_dto.CatalogueMutation which is the parsed DROP QUOTA mutation.
// Returns error when the statement cannot be parsed.
func (p *parser) parseDropQuota() (*querier_dto.CatalogueMutation, error) {
	return p.parseRBACDrop(keywordQuota, querier_dto.MutationDropQuota, engineKeyQuotaName)
}

// parseDropSettingsProfile handles `DROP SETTINGS PROFILE [IF EXISTS] name [, ...]
// [...]`.
//
// Returns *querier_dto.CatalogueMutation which is the parsed DROP SETTINGS PROFILE
// mutation.
// Returns error when the statement cannot be parsed.
func (p *parser) parseDropSettingsProfile() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(keywordDrop)
	p.matchKeyword(keywordSettings)
	return p.captureRBACPolicyOrProfile(keywordProfile, querier_dto.MutationDropSettingsProfile, engineKeyProfileName, false)
}

// parseRBACDrop is the shared body of the RBAC DROP handlers (USER, ROLE, QUOTA).
//
// Takes objectKeyword (string) which is the object word to match after DROP.
// Takes kind (querier_dto.MutationKind) which is the mutation kind to emit.
// Takes nameKey (string) which is the EngineSpecific key under which to store the name.
//
// Returns *querier_dto.CatalogueMutation which is the parsed DROP mutation.
// Returns error when the object keyword is missing or the name cannot be parsed.
func (p *parser) parseRBACDrop(objectKeyword string, kind querier_dto.MutationKind, nameKey string) (*querier_dto.CatalogueMutation, error) {
	return p.parseRBACAlterOrDrop(keywordDrop, objectKeyword, kind, nameKey)
}

// parseRBACAlterOrDrop consolidates the ALTER and DROP RBAC paths.
//
// Both forms use matchIfExists. An optional `ON CLUSTER <name>` clause is captured under
// engineClauseOnCluster before the remainder of the statement is consumed so the cluster
// qualifier is preserved separately from the trailing statement body.
//
// Takes verb (string) which is the leading keyword (ALTER or DROP).
// Takes objectKeyword (string) which is the object word to match after the verb.
// Takes kind (querier_dto.MutationKind) which is the mutation kind to emit.
// Takes nameKey (string) which is the EngineSpecific key under which to store the name.
//
// Returns *querier_dto.CatalogueMutation which is the parsed mutation.
// Returns error when the object keyword is missing or the name cannot be parsed.
func (p *parser) parseRBACAlterOrDrop(verb, objectKeyword string, kind querier_dto.MutationKind, nameKey string) (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword(verb)
	if !p.matchKeyword(objectKeyword) {
		return nil, fmt.Errorf(errExpectedKeyword, objectKeyword, p.current().position)
	}
	p.matchIfExists()
	name, err := p.parseIdentifierOrKeyword()
	if err != nil {
		return nil, err
	}
	engineSpecific := map[string]string{
		nameKey: name,
	}
	if cluster := p.matchOnCluster(); cluster != "" {
		engineSpecific[engineClauseOnCluster] = cluster
	}
	engineSpecific[engineKeyStatementBody] = p.consumeRemainderAsText()
	return &querier_dto.CatalogueMutation{
		Kind:           kind,
		EngineSpecific: engineSpecific,
	}, nil
}

// parseGrant handles `GRANT priv [, ...] ON target TO principal [, ...] [WITH GRANT
// OPTION]`.
//
// Returns *querier_dto.CatalogueMutation which is the parsed GRANT mutation.
// Returns error when the statement cannot be parsed.
func (p *parser) parseGrant() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword("GRANT")
	return &querier_dto.CatalogueMutation{
		Kind: querier_dto.MutationGrantManagement,
		EngineSpecific: map[string]string{
			engineKeyRBACKind:      "GRANT",
			engineKeyStatementBody: p.consumeRemainderAsText(),
		},
	}, nil
}

// parseRevoke handles `REVOKE priv [, ...] ON target FROM principal [, ...]`.
//
// Returns *querier_dto.CatalogueMutation which is the parsed REVOKE mutation.
// Returns error when the statement cannot be parsed.
func (p *parser) parseRevoke() (*querier_dto.CatalogueMutation, error) {
	p.mustKeyword("REVOKE")
	return &querier_dto.CatalogueMutation{
		Kind: querier_dto.MutationGrantManagement,
		EngineSpecific: map[string]string{
			engineKeyRBACKind:      "REVOKE",
			engineKeyStatementBody: p.consumeRemainderAsText(),
		},
	}, nil
}
