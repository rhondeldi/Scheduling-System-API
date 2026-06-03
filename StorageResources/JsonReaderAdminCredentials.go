package StorageResources

import (
	"encoding/json"
	"os"
	"path"

	AdminResource "github.com/mrdcvlsc/scheduling-system-backend/Resources/Admin"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

const adminCredentialsFile = "admin_credentials.json"

func (s *JsonReader) ReadAdminCredentials() (*AdminResource.AdminCredentials, error) {
	projectRoot, errProjectRoot := Utils.FindProjectRoot()
	if errProjectRoot != nil {
		return nil, errProjectRoot
	}

	credentialsFile := path.Join(projectRoot, adminCredentialsFile)
	data, err := os.ReadFile(credentialsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	credentials := &AdminResource.AdminCredentials{}
	if err := json.Unmarshal(data, credentials); err != nil {
		return nil, err
	}

	return credentials, nil
}
