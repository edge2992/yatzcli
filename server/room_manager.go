package server

import (
	"errors"
	"sync"
)

type RoomManager struct {
	rooms map[string]*Room
	mutex sync.RWMutex
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: make(map[string]*Room),
	}
}

func (rm *RoomManager) CreateRoom(roomID string) (*Room, error) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if _, exists := rm.rooms[roomID]; exists {
		return nil, errors.New("Room already exists")
	}
	newRoom := NewRoom(roomID)
	rm.rooms[roomID] = newRoom
	return newRoom, nil
}

func (rm *RoomManager) DestroyRoom(roomID string) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	delete(rm.rooms, roomID)
}

func (rm *RoomManager) JoinRoom(roomID string, player *Player) (*Room, error) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	room, exists := rm.rooms[roomID]
	if !exists {
		return nil, errors.New("Room does not exist")
	}

	if err := room.AddPlayer(player); err != nil {
		return nil, err
	}
	return room, nil
}

func (rm *RoomManager) LeaveRoom(roomID string, player *Player) error {
	room, err := rm.GetRoom(roomID)
	if err != nil {
		return err
	}

	return room.RemovePlayer(player)
}

func (rm *RoomManager) ListRooms() []*Room {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	roomList := make([]*Room, 0, len(rm.rooms))
	for _, room := range rm.rooms {
		roomList = append(roomList, room)
	}
	return roomList
}

func (rm *RoomManager) GetRoom(roomID string) (*Room, error) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	room, exists := rm.rooms[roomID]
	if !exists {
		return nil, errors.New("Room does not exist")
	}
	return room, nil
}

func (rm *RoomManager) StartGame(roomID string, started_randomly bool) error {
	room, err := rm.GetRoom(roomID)
	if err != nil {
		return err
	}

	room.StartGame(started_randomly)
	return nil
}
