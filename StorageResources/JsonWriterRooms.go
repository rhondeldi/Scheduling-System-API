package StorageResources

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"sort"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Rooms"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

func (s *JsonWriter) CreateRoom(new_room Rooms.Room) error {

	if new_room.RoomID != 0 {
		return errors.New("cannot create a new room with a non zero room ID because that would overwrite a room item")
	}

	if new_room.Capacity > uint16(Rooms.MAX_ROOM_CAPACITY) {
		return fmt.Errorf("json.CreateRoom room to add has a capacity of %d section(s), the maximum allowed is only 15", new_room.Capacity)
	}

	RoomMutex.Lock()
	defer RoomMutex.Unlock()

	all_rooms, err_read := json_read_all_rooms()

	if err_read != nil {
		return err_read
	}

	for _, room := range all_rooms {
		if Utils.IsEqualStrCaseInsensitiveIgnoreWhiteSpace(room.Name, new_room.Name) {
			return fmt.Errorf("another room with the name '%s' already exists", new_room.Name)
		}
	}

	new_id_num := uint16(1)

	if len(all_rooms) > 0 {
		new_id_num = all_rooms[len(all_rooms)-1].RoomID + 1
	}

	all_rooms = append(all_rooms, Rooms.Room{
		RoomID:             new_id_num,
		DepartmentID:       new_room.DepartmentID,
		Capacity:           new_room.Capacity,
		RoomType:           new_room.RoomType,
		Name:               new_room.Name,
		SharingDepartments: new_room.SharingDepartments,
	})

	err_save_rooms := json_save_all_rooms(all_rooms)

	if err_save_rooms != nil {
		return err_save_rooms
	}

	return nil
}

func (s *JsonWriter) UpdateRoom(room_to_update Rooms.Room) error {

	if room_to_update.RoomID == 0 {
		return errors.New("parameter argument missing invalid room ID")
	}

	if room_to_update.Capacity > uint16(Rooms.MAX_ROOM_CAPACITY) {
		return fmt.Errorf("json.UpdateRoom room to updated has a capacity of %d section(s), the maximum allowed is only 15", room_to_update.Capacity)
	}

	RoomMutex.Lock()
	defer RoomMutex.Unlock()

	all_rooms, err_read := json_read_all_rooms()

	if err_read != nil {
		return err_read
	}

	for _, room := range all_rooms {
		if room.RoomID == room_to_update.RoomID {
			continue
		}

		if Utils.IsEqualStrCaseInsensitiveIgnoreWhiteSpace(room.Name, room_to_update.Name) {
			return fmt.Errorf("another room with the name '%s' already exists", room_to_update.Name)
		}
	}

	has_id := false
	to_update_idx := -1

	for idx, room := range all_rooms {
		if room.RoomID == room_to_update.RoomID {
			has_id = true
			to_update_idx = idx
			break
		}
	}

	if !has_id {
		return errors.New("room to update does not exist in the json file")
	}

	all_rooms[to_update_idx] = Rooms.Room{
		RoomID:             room_to_update.RoomID,
		DepartmentID:       room_to_update.DepartmentID,
		Capacity:           room_to_update.Capacity,
		RoomType:           room_to_update.RoomType,
		Name:               room_to_update.Name,
		SharingDepartments: room_to_update.SharingDepartments,
	}

	err_save_rooms := json_save_all_rooms(all_rooms)

	if err_save_rooms != nil {
		return err_save_rooms
	}

	return nil
}

func (s *JsonWriter) DeleteRoom(room_id uint16) error {

	if room_id == 0 {
		return errors.New("parameter argument missing invalid room ID")
	}

	RoomMutex.Lock()
	defer RoomMutex.Unlock()

	all_rooms, err_read := json_read_all_rooms()

	if err_read != nil {
		return err_read
	}

	rooms_deleted := make([]Rooms.Room, 0)

	has_id := false

	for _, room := range all_rooms {
		if room.RoomID == room_id {
			has_id = true
		} else {
			rooms_deleted = append(rooms_deleted, room)
		}
	}

	if !has_id {
		return errors.New("room to delete does not exist in the json file")
	}

	err_save_rooms := json_save_all_rooms(rooms_deleted)

	if err_save_rooms != nil {
		return err_save_rooms
	}

	return nil
}

func json_save_all_rooms(rooms []Rooms.Room) error {
	project_root, err_project_root := Utils.FindProjectRoot()

	if err_project_root != nil {
		return err_project_root
	}

	rooms_json_file := path.Join(project_root, "scheduling-system-temporary-data", "rooms.json")

	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].RoomID < rooms[j].RoomID
	})

	rooms_byte_data, err := json.MarshalIndent(rooms, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(rooms_json_file, rooms_byte_data, 0644); err != nil {
		return err
	}

	return nil
}
