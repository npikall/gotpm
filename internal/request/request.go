package request

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/charmbracelet/log"
	"github.com/go-playground/validator/v10"
	"github.com/npikall/gotpm/internal/bump"
)

const TypstPackageEndpoint string = "https://api.github.com/repos/typst/packages/contents/packages/preview/"

type ResponseModel struct {
	Name string `json:"name" validate:"semver"`
}

func FetchDataFromGitHub(url string, ctx context.Context) ([]*ResponseModel, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer closeResponse(resp)

	var result []*ResponseModel
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

var ErrInvalidValidation = errors.New("invalid validation")

// Validate that a given response from the github api of the typst
// package repository only contains valid semver strings.
func ValidateVersion(resp ResponseModel) (bool, error) {
	validate := validator.New()
	err := validate.Struct(resp)

	if err != nil {
		if _, ok := err.(validator.ValidationErrors); ok {
			return false, nil
		}

		if _, ok := err.(*validator.InvalidValidationError); ok {
			return false, ErrInvalidValidation
		}
	}
	return true, nil
}

func GetLatestVersion(versions []*ResponseModel) (string, error) {
	candidate := bump.NewVersion()
	for _, v := range versions {
		currentVersion, err := bump.ParseVersion(v.Name)
		if err != nil {
			return "", err
		}
		res := bump.CompareVersions(candidate, currentVersion)
		switch res {
		case -1:
			candidate = currentVersion
		default:
			continue
		}
	}
	return candidate.String(), nil
}

func closeResponse(resp *http.Response) {
	err := resp.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
}
