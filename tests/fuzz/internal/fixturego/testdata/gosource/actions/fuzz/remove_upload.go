package fuzz

import (
	"fmt"

	"piko.sh/piko"
	"piko.sh/piko/wdk/storage"
)

type RemoveUploadAction struct{ piko.ActionMetadata }
type RemoveUploadInput struct{ Key string }
type RemoveUploadResult struct{ Removed string }

func (a *RemoveUploadAction) Call(in RemoveUploadInput) (RemoveUploadResult, error) {
	if in.Key != "payload.bin" {
		return RemoveUploadResult{}, fmt.Errorf("unexpected fixture key %q", in.Key)
	}
	service, err := uploadService()
	if err != nil {
		return RemoveUploadResult{}, err
	}
	err = service.RemoveObject(a.Ctx(), "fuzz-upload", storage.GetParams{Repository: "uploads", Key: in.Key})
	if err != nil {
		return RemoveUploadResult{}, err
	}
	return RemoveUploadResult{Removed: in.Key}, nil
}
