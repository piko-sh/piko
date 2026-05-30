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

package db_engine_mysql

const (
	// keywordAFTER names the SQL AFTER keyword.
	keywordAFTER = "AFTER"

	// keywordALL names the SQL ALL keyword.
	keywordALL = "ALL"

	// keywordAND names the SQL AND keyword.
	keywordAND = "AND"

	// keywordAS names the SQL AS keyword.
	keywordAS = "AS"

	// keywordASC names the SQL ASC keyword.
	keywordASC = "ASC"

	// keywordAUTO names the SQL AUTO_INCREMENT keyword.
	keywordAUTO = "AUTO_INCREMENT"

	// keywordBY names the SQL BY keyword.
	keywordBY = "BY"

	// keywordCASCADE names the SQL CASCADE keyword.
	keywordCASCADE = "CASCADE"

	// keywordCHARSET names the SQL CHARSET keyword.
	keywordCHARSET = "CHARSET"

	// keywordCHECK names the SQL CHECK keyword.
	keywordCHECK = "CHECK"

	// keywordCOLLATE names the SQL COLLATE keyword.
	keywordCOLLATE = "COLLATE"

	// keywordCOLUMN names the SQL COLUMN keyword.
	keywordCOLUMN = "COLUMN"

	// keywordCOMMENT names the SQL COMMENT keyword.
	keywordCOMMENT = "COMMENT"

	// keywordCONSTRAINT names the SQL CONSTRAINT keyword.
	keywordCONSTRAINT = "CONSTRAINT"

	// keywordCREATE names the SQL CREATE keyword.
	keywordCREATE = "CREATE"

	// keywordCURRENT names the SQL CURRENT keyword.
	keywordCURRENT = "CURRENT"

	// keywordDATA names the SQL DATA keyword.
	keywordDATA = "DATA"

	// keywordDATABASE names the SQL DATABASE keyword.
	keywordDATABASE = "DATABASE"

	// keywordDEFAULT names the SQL DEFAULT keyword.
	keywordDEFAULT = "DEFAULT"

	// keywordDEFINER names the SQL DEFINER keyword.
	keywordDEFINER = "DEFINER"

	// keywordDESC names the SQL DESC keyword.
	keywordDESC = "DESC"

	// keywordDETERMINISTIC names the SQL DETERMINISTIC keyword.
	keywordDETERMINISTIC = "DETERMINISTIC"

	// keywordDROP names the SQL DROP keyword.
	keywordDROP = "DROP"

	// keywordDUPLICATE names the SQL DUPLICATE keyword.
	keywordDUPLICATE = "DUPLICATE"

	// keywordENGINE names the SQL ENGINE keyword.
	keywordENGINE = "ENGINE"

	// keywordEXCEPT names the SQL EXCEPT keyword.
	keywordEXCEPT = "EXCEPT"

	// keywordEND names the SQL END keyword.
	keywordEND = "END"

	// keywordEXISTS names the SQL EXISTS keyword.
	keywordEXISTS = "EXISTS"

	// keywordFIRST names the SQL FIRST keyword.
	keywordFIRST = "FIRST"

	// keywordFOR names the SQL FOR keyword.
	keywordFOR = "FOR"

	// keywordFOREIGN names the SQL FOREIGN keyword.
	keywordFOREIGN = "FOREIGN"

	// keywordFROM names the SQL FROM keyword.
	keywordFROM = "FROM"

	// keywordFUNCTION names the SQL FUNCTION keyword.
	keywordFUNCTION = "FUNCTION"

	// keywordGROUP names the SQL GROUP keyword.
	keywordGROUP = "GROUP"

	// keywordHAVING names the SQL HAVING keyword.
	keywordHAVING = "HAVING"

	// keywordIGNORE names the SQL IGNORE keyword.
	keywordIGNORE = "IGNORE"

	// keywordINDEX names the SQL INDEX keyword.
	keywordINDEX = "INDEX"

	// keywordINTERSECT names the SQL INTERSECT keyword.
	keywordINTERSECT = "INTERSECT"

	// keywordJOIN names the SQL JOIN keyword.
	keywordJOIN = "JOIN"

	// keywordKEY names the SQL KEY keyword.
	keywordKEY = "KEY"

	// keywordLANGUAGE names the SQL LANGUAGE keyword.
	keywordLANGUAGE = "LANGUAGE"

	// keywordLAST names the SQL LAST keyword.
	keywordLAST = "LAST"

	// keywordLATERAL names the SQL LATERAL keyword.
	keywordLATERAL = "LATERAL"

	// keywordLIMIT names the SQL LIMIT keyword.
	keywordLIMIT = "LIMIT"

	// keywordMODIFIES names the SQL MODIFIES keyword.
	keywordMODIFIES = "MODIFIES"

	// keywordNOT names the SQL NOT keyword.
	keywordNOT = "NOT"

	// keywordNULL names the SQL NULL keyword.
	keywordNULL = "NULL"

	// keywordNULLS names the SQL NULLS keyword.
	keywordNULLS = "NULLS"

	// keywordOFFSET names the SQL OFFSET keyword.
	keywordOFFSET = "OFFSET"

	// keywordNO names the SQL NO keyword.
	keywordNO = "NO"

	// keywordON names the SQL ON keyword.
	keywordON = "ON"

	// keywordORDER names the SQL ORDER keyword.
	keywordORDER = "ORDER"

	// keywordPRIMARY names the SQL PRIMARY keyword.
	keywordPRIMARY = "PRIMARY"

	// keywordPROCEDURE names the SQL PROCEDURE keyword.
	keywordPROCEDURE = "PROCEDURE"

	// keywordREADS names the SQL READS keyword.
	keywordREADS = "READS"

	// keywordREPLACE names the SQL REPLACE keyword.
	keywordREPLACE = "REPLACE"

	// keywordRESTRICT names the SQL RESTRICT keyword.
	keywordRESTRICT = "RESTRICT"

	// keywordRETURN names the SQL RETURN keyword.
	keywordRETURN = "RETURN"

	// keywordRETURNING names the SQL RETURNING keyword.
	keywordRETURNING = "RETURNING"

	// keywordRETURNS names the SQL RETURNS keyword.
	keywordRETURNS = "RETURNS"

	// keywordROLLUP names the SQL ROLLUP keyword.
	keywordROLLUP = "ROLLUP"

	// keywordSCHEMA names the SQL SCHEMA keyword.
	keywordSCHEMA = "SCHEMA"

	// keywordSECURITY names the SQL SECURITY keyword.
	keywordSECURITY = "SECURITY"

	// keywordIN names the SQL IN keyword.
	keywordIN = "IN"

	// keywordCAST names the SQL CAST keyword.
	keywordCAST = "CAST"

	// keywordSELECT names the SQL SELECT keyword.
	keywordSELECT = "SELECT"

	// keywordSEPARATOR names the SQL SEPARATOR keyword.
	keywordSEPARATOR = "SEPARATOR"

	// keywordSQL names the SQL SQL keyword.
	keywordSQL = "SQL"

	// keywordSET names the SQL SET keyword.
	keywordSET = "SET"

	// keywordTABLE names the SQL TABLE keyword.
	keywordTABLE = "TABLE"

	// keywordUNION names the SQL UNION keyword.
	keywordUNION = "UNION"

	// keywordUNIQUE names the SQL UNIQUE keyword.
	keywordUNIQUE = "UNIQUE"

	// keywordUNSIGNED names the SQL UNSIGNED keyword.
	keywordUNSIGNED = "UNSIGNED"

	// keywordUSING names the SQL USING keyword.
	keywordUSING = "USING"

	// keywordVALUES names the SQL VALUES keyword.
	keywordVALUES = "VALUES"

	// keywordWHERE names the SQL WHERE keyword.
	keywordWHERE = "WHERE"

	// keywordWITH names the SQL WITH keyword.
	keywordWITH = "WITH"

	// decimalBase is the base used when parsing decimal integer literals.
	decimalBase = 10
)
