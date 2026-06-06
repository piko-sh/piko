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

package db_engine_sqlite

const (
	// keywordAS is the SQL AS keyword used for aliases.
	keywordAS = "AS"

	// keywordAND is the SQL AND boolean operator.
	keywordAND = "AND"

	// keywordBY is the SQL BY clause keyword used after GROUP and ORDER.
	keywordBY = "BY"

	// keywordCHECK is the SQL CHECK constraint keyword.
	keywordCHECK = "CHECK"

	// keywordCOLLATE is the SQL COLLATE keyword.
	keywordCOLLATE = "COLLATE"

	// keywordCONFLICT is the SQL ON CONFLICT clause keyword.
	keywordCONFLICT = "CONFLICT"

	// keywordCONSTRAINT is the SQL CONSTRAINT keyword.
	keywordCONSTRAINT = "CONSTRAINT"

	// keywordCREATE is the SQL CREATE DDL keyword.
	keywordCREATE = "CREATE"

	// keywordDROP is the SQL DROP DDL keyword.
	keywordDROP = "DROP"

	// keywordEXCEPT is the SQL EXCEPT compound operator keyword.
	keywordEXCEPT = "EXCEPT"

	// keywordEXISTS is the SQL EXISTS predicate keyword.
	keywordEXISTS = "EXISTS"

	// keywordFROM is the SQL FROM clause keyword.
	keywordFROM = "FROM"

	// keywordGROUP is the SQL GROUP clause keyword.
	keywordGROUP = "GROUP"

	// keywordHAVING is the SQL HAVING clause keyword.
	keywordHAVING = "HAVING"

	// keywordIF is the SQL IF keyword used in IF EXISTS clauses.
	keywordIF = "IF"

	// keywordINTERSECT is the SQL INTERSECT compound operator keyword.
	keywordINTERSECT = "INTERSECT"

	// keywordJOIN is the SQL JOIN clause keyword.
	keywordJOIN = "JOIN"

	// keywordKEY is the SQL KEY keyword used in PRIMARY KEY and FOREIGN KEY.
	keywordKEY = "KEY"

	// keywordLIMIT is the SQL LIMIT clause keyword.
	keywordLIMIT = "LIMIT"

	// keywordNOT is the SQL NOT boolean operator.
	keywordNOT = "NOT"

	// keywordON is the SQL ON join condition keyword.
	keywordON = "ON"

	// keywordOR is the SQL OR boolean operator.
	keywordOR = "OR"

	// keywordORDER is the SQL ORDER clause keyword.
	keywordORDER = "ORDER"

	// keywordPRIMARY is the SQL PRIMARY KEY keyword.
	keywordPRIMARY = "PRIMARY"

	// keywordREFERENCES is the SQL REFERENCES foreign key keyword.
	keywordREFERENCES = "REFERENCES"

	// keywordRETURNING is the SQL RETURNING clause keyword.
	keywordRETURNING = "RETURNING"

	// keywordSELECT is the SQL SELECT statement keyword.
	keywordSELECT = "SELECT"

	// keywordSET is the SQL SET clause keyword used in UPDATE statements.
	keywordSET = "SET"

	// keywordTABLE is the SQL TABLE DDL keyword.
	keywordTABLE = "TABLE"

	// keywordTRIGGER is the SQL TRIGGER DDL keyword.
	keywordTRIGGER = "TRIGGER"

	// keywordUNION is the SQL UNION compound operator keyword.
	keywordUNION = "UNION"

	// keywordUNIQUE is the SQL UNIQUE constraint keyword.
	keywordUNIQUE = "UNIQUE"

	// keywordVALUES is the SQL VALUES clause keyword used in INSERT.
	keywordVALUES = "VALUES"

	// keywordWHERE is the SQL WHERE clause keyword.
	keywordWHERE = "WHERE"

	// keywordWITH is the SQL WITH common-table-expression keyword.
	keywordWITH = "WITH"

	// minTokensForClassification is the minimum token count required before statement
	// classification inspects the second keyword.
	minTokensForClassification = 3

	// doubleArrowOperatorLength is the byte length of the `->>` JSON operator.
	doubleArrowOperatorLength = 3
)
