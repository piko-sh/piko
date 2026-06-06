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

package db_engine_duckdb

const (
	// keywordALL holds the SQL keyword "ALL".
	keywordALL = "ALL"

	// keywordAND holds the SQL keyword "AND".
	keywordAND = "AND"

	// keywordAS holds the SQL keyword "AS".
	keywordAS = "AS"

	// keywordASC holds the SQL keyword "ASC".
	keywordASC = "ASC"

	// keywordBY holds the SQL keyword "BY".
	keywordBY = "BY"

	// keywordCASCADE holds the SQL keyword "CASCADE".
	keywordCASCADE = "CASCADE"

	// keywordCHECK holds the SQL keyword "CHECK".
	keywordCHECK = "CHECK"

	// keywordCOLUMN holds the SQL keyword "COLUMN".
	keywordCOLUMN = "COLUMN"

	// keywordCONSTRAINT holds the SQL keyword "CONSTRAINT".
	keywordCONSTRAINT = "CONSTRAINT"

	// keywordCREATE holds the SQL keyword "CREATE".
	keywordCREATE = "CREATE"

	// keywordCURRENT holds the SQL keyword "CURRENT".
	keywordCURRENT = "CURRENT"

	// keywordDEFAULT holds the SQL keyword "DEFAULT".
	keywordDEFAULT = "DEFAULT"

	// keywordDESC holds the SQL keyword "DESC".
	keywordDESC = "DESC"

	// keywordDROP holds the SQL keyword "DROP".
	keywordDROP = "DROP"

	// keywordEXCEPT holds the SQL keyword "EXCEPT".
	keywordEXCEPT = "EXCEPT"

	// keywordEXISTS holds the SQL keyword "EXISTS".
	keywordEXISTS = "EXISTS"

	// keywordFETCH holds the SQL keyword "FETCH".
	keywordFETCH = "FETCH"

	// keywordFIRST holds the SQL keyword "FIRST".
	keywordFIRST = "FIRST"

	// keywordFOR holds the SQL keyword "FOR".
	keywordFOR = "FOR"

	// keywordFROM holds the SQL keyword "FROM".
	keywordFROM = "FROM"

	// keywordGROUP holds the SQL keyword "GROUP".
	keywordGROUP = "GROUP"

	// keywordHAVING holds the SQL keyword "HAVING".
	keywordHAVING = "HAVING"

	// keywordINSTALL holds the SQL keyword "INSTALL".
	keywordINSTALL = "INSTALL"

	// keywordINTERSECT holds the SQL keyword "INTERSECT".
	keywordINTERSECT = "INTERSECT"

	// keywordJOIN holds the SQL keyword "JOIN".
	keywordJOIN = "JOIN"

	// keywordKEY holds the SQL keyword "KEY".
	keywordKEY = "KEY"

	// keywordLAST holds the SQL keyword "LAST".
	keywordLAST = "LAST"

	// keywordLATERAL holds the SQL keyword "LATERAL".
	keywordLATERAL = "LATERAL"

	// keywordLIMIT holds the SQL keyword "LIMIT".
	keywordLIMIT = "LIMIT"

	// keywordLOAD holds the SQL keyword "LOAD".
	keywordLOAD = "LOAD"

	// keywordMACRO holds the SQL keyword "MACRO".
	keywordMACRO = "MACRO"

	// keywordNOT holds the SQL keyword "NOT".
	keywordNOT = "NOT"

	// keywordIN holds the SQL keyword "IN".
	keywordIN = "IN"

	// keywordCAST holds the SQL keyword "CAST".
	keywordCAST = "CAST"

	// keywordIS holds the SQL keyword "IS".
	keywordIS = "IS"

	// keywordNULL holds the SQL keyword "NULL".
	keywordNULL = "NULL"

	// keywordNULLS holds the SQL keyword "NULLS".
	keywordNULLS = "NULLS"

	// keywordOFFSET holds the SQL keyword "OFFSET".
	keywordOFFSET = "OFFSET"

	// keywordON holds the SQL keyword "ON".
	keywordON = "ON"

	// keywordORDER holds the SQL keyword "ORDER".
	keywordORDER = "ORDER"

	// keywordPIVOT holds the SQL keyword "PIVOT".
	keywordPIVOT = "PIVOT"

	// keywordPOSITIONAL holds the SQL keyword "POSITIONAL".
	keywordPOSITIONAL = "POSITIONAL"

	// keywordPRIMARY holds the SQL keyword "PRIMARY".
	keywordPRIMARY = "PRIMARY"

	// keywordQUALIFY holds the SQL keyword "QUALIFY".
	keywordQUALIFY = "QUALIFY"

	// keywordRESTRICT holds the SQL keyword "RESTRICT".
	keywordRESTRICT = "RESTRICT"

	// keywordRETURNING holds the SQL keyword "RETURNING".
	keywordRETURNING = "RETURNING"

	// keywordROW holds the SQL keyword "ROW".
	keywordROW = "ROW"

	// keywordROWS holds the SQL keyword "ROWS".
	keywordROWS = "ROWS"

	// keywordSCHEMA holds the SQL keyword "SCHEMA".
	keywordSCHEMA = "SCHEMA"

	// keywordSELECT holds the SQL keyword "SELECT".
	keywordSELECT = "SELECT"

	// keywordSET holds the SQL keyword "SET".
	keywordSET = "SET"

	// keywordTABLE holds the SQL keyword "TABLE".
	keywordTABLE = "TABLE"

	// keywordTIME holds the SQL keyword "TIME".
	keywordTIME = "TIME"

	// keywordTYPE holds the SQL keyword "TYPE".
	keywordTYPE = "TYPE"

	// keywordUNION holds the SQL keyword "UNION".
	keywordUNION = "UNION"

	// keywordUNIQUE holds the SQL keyword "UNIQUE".
	keywordUNIQUE = "UNIQUE"

	// keywordUNPIVOT holds the SQL keyword "UNPIVOT".
	keywordUNPIVOT = "UNPIVOT"

	// keywordUSING holds the SQL keyword "USING".
	keywordUSING = "USING"

	// keywordVALUES holds the SQL keyword "VALUES".
	keywordVALUES = "VALUES"

	// keywordWHERE holds the SQL keyword "WHERE".
	keywordWHERE = "WHERE"

	// keywordWITH holds the SQL keyword "WITH".
	keywordWITH = "WITH"

	// keywordZONE holds the SQL keyword "ZONE".
	keywordZONE = "ZONE"

	// arraySubscriptSuffix is appended to a type name to mark it as an array.
	arraySubscriptSuffix = "[]"

	// fallbackListEngineName is the engine name used when a list element type cannot be
	// resolved.
	fallbackListEngineName = "list"

	// maxTypeModifierValue caps an accepted numeric type modifier. DuckDB stores type
	// modifiers as a signed 32-bit value, so any larger figure is already out of range;
	// modifiers above this ceiling (or below their minimum) are treated as absent rather
	// than written into the catalogue.
	maxTypeModifierValue = 1<<31 - 1
)
