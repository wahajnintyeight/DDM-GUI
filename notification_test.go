package main

import "testing"

func TestCompletionNotification(t *testing.T) {
	for _, test := range []struct {
		update DownloadUpdate
		want   string
	}{
		{DownloadUpdate{State: "complete", Progress: 100, Message: "Download complete"}, "Download complete"},
		{DownloadUpdate{State: "complete", Progress: 100, Message: "Downgrade complete"}, "Downgrade complete"},
		{DownloadUpdate{State: "running", Progress: 100, Message: "Downloading files"}, ""},
		{DownloadUpdate{State: "error", Progress: 100, Message: "Verification failed"}, ""},
		{DownloadUpdate{State: "cancelled", Progress: 100, Message: "Download cancelled"}, ""},
	} {
		if got := completionNotification(test.update); got != test.want {
			t.Errorf("%s: got %q, want %q", test.update.State, got, test.want)
		}
	}
}
