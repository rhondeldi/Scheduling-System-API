package StorageResources

import (
	"encoding/json"
	"os"
	"path"

	AdminResource "github.com/mrdcvlsc/scheduling-system-backend/Resources/Admin"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

func (s *JsonWriter) UpsertAdminCredentials(credentials AdminResource.AdminCredentials) error {
	projectRoot, errProjectRoot := Utils.FindProjectRoot()
	if errProjectRoot != nil {
		return errProjectRoot
	}

	credentialsFile := path.Join(projectRoot, adminCredentialsFile)
	saveCredentials := credentials
	if saveCredentials.PasswordHash != "" {
		saveCredentials.Password = ""
	}

	data, err := json.MarshalIndent(saveCredentials, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(credentialsFile, data, 0600)
}
