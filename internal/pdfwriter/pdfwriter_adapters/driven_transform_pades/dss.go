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

package driven_transform_pades

import (
	"errors"
	"fmt"

	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/pdfparse"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

// addDSSDictionary creates a Document Security Store (DSS) dictionary
// in the catalog containing pre-fetched OCSP responses and CRLs for
// long-term validation (PAdES B-LT and above). The DSS follows the
// PDF 2.0 / ETSI EN 319 142-1 specification.
//
// Each OCSP response and CRL is stored as a separate stream object.
// The DSS dictionary references these streams via /OCSPs and /CRLs
// arrays.
//
// Takes writer (*pdfparse.Writer) which is the PDF document writer.
// Takes opts (*pdfwriter_dto.PadesSignOptions) which provides the
// pre-fetched OCSP responses and CRLs.
//
// Returns error when the catalog cannot be located or updated.
func addDSSDictionary(writer *pdfparse.Writer, opts *pdfwriter_dto.PadesSignOptions) error {
	if len(opts.OCSPResponses) == 0 && len(opts.CRLs) == 0 {
		return nil
	}

	dssPairs := []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("DSS")},
	}

	if len(opts.OCSPResponses) > 0 {
		ocspRefs := make([]pdfparse.Object, 0, len(opts.OCSPResponses))
		for _, ocspData := range opts.OCSPResponses {
			objNum := writer.AddObject(pdfparse.StreamObj(pdfparse.Dict{}, ocspData))
			ocspRefs = append(ocspRefs, pdfparse.RefObj(objNum, 0))
		}
		dssPairs = append(dssPairs, pdfparse.DictPair{
			Key: "OCSPs", Value: pdfparse.Arr(ocspRefs...),
		})
	}

	if len(opts.CRLs) > 0 {
		crlRefs := make([]pdfparse.Object, 0, len(opts.CRLs))
		for _, crlData := range opts.CRLs {
			objNum := writer.AddObject(pdfparse.StreamObj(pdfparse.Dict{}, crlData))
			crlRefs = append(crlRefs, pdfparse.RefObj(objNum, 0))
		}
		dssPairs = append(dssPairs, pdfparse.DictPair{
			Key: "CRLs", Value: pdfparse.Arr(crlRefs...),
		})
	}

	dssObjNum := writer.AddObject(pdfparse.DictObj(pdfparse.Dict{Pairs: dssPairs}))

	trailer := writer.Trailer()
	rootRef := trailer.GetRef("Root")
	if rootRef.Number == 0 {
		return errors.New("pades-sign: trailer has no /Root reference for DSS")
	}

	catalogObj := writer.GetObject(rootRef.Number)
	catalogDict, ok := catalogObj.Value.(pdfparse.Dict)
	if !ok {
		return fmt.Errorf("pades-sign: catalog object %d is not a dictionary", rootRef.Number)
	}

	catalogDict.Set("DSS", pdfparse.RefObj(dssObjNum, 0))
	writer.SetObject(rootRef.Number, pdfparse.DictObj(catalogDict))
	return nil
}
