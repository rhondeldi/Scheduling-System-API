package StorageResources

import (
	"encoding/json"
	"os"
	"path"
	"sort"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Rooms"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

func (s *JsonReader) ReadAllRooms() ([]Rooms.Room, error) {

	RoomMutex.Lock()
	defer RoomMutex.Unlock()

	return json_read_all_rooms()
}

func json_read_all_rooms() ([]Rooms.Room, error) {

	project_root, err_project_root := Utils.FindProjectRoot()

	if err_project_root != nil {
		return nil, err_project_root
	}

	rooms_json_file := path.Join(project_root, "scheduling-system-temporary-data", "rooms.json")
	rooms_byte_data, err := os.ReadFile(rooms_json_file)

	if err != nil {
		return nil, err
	}

	rooms := make([]Rooms.Room, 0)
	err = json.Unmarshal(rooms_byte_data, &rooms)

	if err != nil {
		return nil, err
	}

	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].RoomID < rooms[j].RoomID
	})

	return rooms, nil
}

// return a nil room if room does not exist
func (s *JsonReader) ReadRoom(room_id uint16) (*Rooms.Room, error) {

	rooms, err := s.ReadAllRooms()

	if err != nil {
		return nil, err
	}

	for _, room := range rooms {
		if room.RoomID == room_id {
			return &room, nil
		}
	}

	return nil, nil
}
