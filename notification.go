package main

import (
	"log"

	toast "git.sr.ht/~jackmordaunt/go-toast/v2"
)

func completionNotification(update DownloadUpdate) string {
	if update.State != "complete" {
		return ""
	}
	return update.Message
}

func showCompletionNotification(message string) {
	if err := toast.SetAppData(toast.AppData{AppID: "DepotDrop", GUID: "{3C67E84D-52C8-49E3-B34F-1A8C37E9F1D2}"}); err != nil {
		log.Printf("register notification: %v", err)
		return
	}
	notification := toast.Notification{AppID: "DepotDrop", Title: "DepotDrop", Body: message}
	if err := notification.Push(); err != nil {
		log.Printf("show notification: %v", err)
	}
}
