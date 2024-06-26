package foodme

import (
	"sync"
	"time"
)

type State struct {
	Connections map[string]Connection
	Mutex       sync.RWMutex
}
type Connection struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

func (c *Connection) IsAlive() bool {
	return time.Now().Unix() < c.ExpiresIn
}

var GlobalState = &State{Connections: make(map[string]Connection), Mutex: sync.RWMutex{}}

func (s *State) AddConnection(username string, accessToken, refreshToken string, lifetime int) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	s.Connections[username] = Connection{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    time.Now().Add(time.Duration(lifetime) * time.Second).Unix(),
	}
}

func (s *State) GetTokens(username string) (accessToken, refreshToken string) {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()
	connection, ok := s.Connections[username]
	if !ok {
		return "", ""
	}
	if !connection.IsAlive() {
		return "", ""
	}
	return connection.AccessToken, connection.RefreshToken
}

func (s *State) GetExpiredUsernames() []string {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()
	usernames := make([]string, 0)
	for username, connection := range s.Connections {
		if !connection.IsAlive() {
			usernames = append(usernames, username)
		}
	}
	return usernames
}

func (s *State) DeleteConnection(username string) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	if _, ok := s.Connections[username]; !ok {
		return
	}
	delete(s.Connections, username)
}
