package main

import (
	"bufio"
	"os"
	"sync"
	"time"
)

var (
	allowedUsers     map[string]struct{}
	allowedUsersLock sync.RWMutex
	lastModified     time.Time
)

func init() {
	allowedUsers = make(map[string]struct{})
}

func isAllowedUser(user string) bool {
	if updateAllowedUsersIfNeeded() {
		allowedUsersLock.RLock()
		defer allowedUsersLock.RUnlock()

		if _, ok := allowedUsers[user]; ok {
			return true
		}
	}
	return false
}

const fileAllowedUsers = "allowed_users.txt"

func updateAllowedUsersIfNeeded() bool {
	file, err := os.Open(fileAllowedUsers)
	if err != nil {
		return false
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return false
	}
	modifiedTime := fileInfo.ModTime()
	if modifiedTime.After(lastModified) {
		allowedUsersLock.Lock()
		defer allowedUsersLock.Unlock()

		scanner := bufio.NewScanner(file)
		tempAllowedUsers := make(map[string]struct{})

		for scanner.Scan() {
			tempAllowedUsers[scanner.Text()] = struct{}{}
		}
		if err := scanner.Err(); err != nil {
			return false
		}
		allowedUsers = tempAllowedUsers
		lastModified = modifiedTime
		return true
	}
	return true
}
