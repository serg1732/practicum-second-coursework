package tui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
)

const (
	shortCommandTimeout  = 3 * time.Second
	commonCommandTimeout = 10 * time.Second
	writeCommandTimeout  = 30 * time.Second
	fileCommandTimeout   = 60 * time.Second
)

type authOKMsg struct{ snapshot SyncSnapshot }
type syncedMsg struct{ snapshot SyncSnapshot }
type savedMsg struct{ snapshot SyncSnapshot }
type deletedMsg struct{ snapshot SyncSnapshot }
type fileSyncedMsg struct{ snapshot SyncSnapshot }
type fileUploadedMsg struct{ snapshot SyncSnapshot }
type fileDownloadedMsg struct{ snapshot SyncSnapshot }
type localFileDeletedMsg struct{ snapshot SyncSnapshot }
type errMsg struct{ err error }

func loginCmd(parent context.Context, api Client, username, password string) tea.Cmd {
	return withSnapshot(parent, commonCommandTimeout, func(ctx context.Context) (SyncSnapshot, error) {
		if err := api.Login(ctx, username, password); err != nil {
			return SyncSnapshot{}, err
		}

		return api.Sync(ctx)
	}, func(snapshot SyncSnapshot) tea.Msg {
		return authOKMsg{snapshot: snapshot}
	})
}

func registerCmd(parent context.Context, api Client, username, password string) tea.Cmd {
	return withSnapshot(parent, commonCommandTimeout, func(ctx context.Context) (SyncSnapshot, error) {
		if err := api.Register(ctx, username, password); err != nil {
			return SyncSnapshot{}, err
		}
		if err := api.Login(ctx, username, password); err != nil {
			return SyncSnapshot{}, err
		}

		return api.Sync(ctx)
	}, func(snapshot SyncSnapshot) tea.Msg {
		return authOKMsg{snapshot: snapshot}
	})
}

func syncCmd(parent context.Context, api Client) tea.Cmd {
	return withSnapshot(parent, commonCommandTimeout, api.Sync, func(snapshot SyncSnapshot) tea.Msg {
		return syncedMsg{snapshot: snapshot}
	})
}

func saveCmd(parent context.Context, api Client, id string, draft Draft) tea.Cmd {
	return withSnapshot(parent, writeCommandTimeout, func(ctx context.Context) (SyncSnapshot, error) {
		var err error
		if id == "" {
			err = api.Create(ctx, draft)
		} else {
			err = api.Update(ctx, id, draft)
		}
		if err != nil {
			return SyncSnapshot{}, err
		}

		return api.Sync(ctx)
	}, func(snapshot SyncSnapshot) tea.Msg {
		return savedMsg{snapshot: snapshot}
	})
}

func deleteCmd(parent context.Context, api Client, id string) tea.Cmd {
	return withSnapshot(parent, writeCommandTimeout, func(ctx context.Context) (SyncSnapshot, error) {
		if err := api.Delete(ctx, id); err != nil {
			return SyncSnapshot{}, err
		}

		return api.Sync(ctx)
	}, func(snapshot SyncSnapshot) tea.Msg {
		return deletedMsg{snapshot: snapshot}
	})
}

func syncFilesCmd(parent context.Context, api Client) tea.Cmd {
	return withSnapshot(parent, fileCommandTimeout, api.SyncFiles, func(snapshot SyncSnapshot) tea.Msg {
		return fileSyncedMsg{snapshot: snapshot}
	})
}

func uploadFileCmd(parent context.Context, api Client, path string) tea.Cmd {
	return withSnapshot(parent, fileCommandTimeout, func(ctx context.Context) (SyncSnapshot, error) {
		if err := api.UploadFile(ctx, path); err != nil {
			return SyncSnapshot{}, err
		}

		return api.Sync(ctx)
	}, func(snapshot SyncSnapshot) tea.Msg {
		return fileUploadedMsg{snapshot: snapshot}
	})
}

func downloadFileCmd(parent context.Context, api Client, name string) tea.Cmd {
	return withSnapshot(parent, fileCommandTimeout, func(ctx context.Context) (SyncSnapshot, error) {
		if err := api.DownloadFile(ctx, name); err != nil {
			return SyncSnapshot{}, err
		}

		return api.Sync(ctx)
	}, func(snapshot SyncSnapshot) tea.Msg {
		return fileDownloadedMsg{snapshot: snapshot}
	})
}

func deleteLocalFileCmd(parent context.Context, api Client, name string) tea.Cmd {
	return withSnapshot(parent, writeCommandTimeout, func(ctx context.Context) (SyncSnapshot, error) {
		if err := api.DeleteLocalFile(ctx, name); err != nil {
			return SyncSnapshot{}, err
		}

		return api.Sync(ctx)
	}, func(snapshot SyncSnapshot) tea.Msg {
		return localFileDeletedMsg{snapshot: snapshot}
	})
}

func removeFileCmd(parent context.Context, api Client, name string) tea.Cmd {
	return withSnapshot(parent, writeCommandTimeout, func(ctx context.Context) (SyncSnapshot, error) {
		if err := api.RemoveFile(ctx, name); err != nil {
			return SyncSnapshot{}, err
		}

		return api.Sync(ctx)
	}, func(snapshot SyncSnapshot) tea.Msg {
		return fileSyncedMsg{snapshot: snapshot}
	})
}

func logoutCmd(parent context.Context, api Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(parent, shortCommandTimeout)
		defer cancel()

		_ = api.Logout(ctx)
		return nil
	}
}

func withSnapshot(
	parent context.Context,
	timeout time.Duration,
	action func(context.Context) (SyncSnapshot, error),
	ok func(SyncSnapshot) tea.Msg,
) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(parent, timeout)
		defer cancel()

		snapshot, err := action(ctx)
		if err != nil {
			return errMsg{err: err}
		}

		return ok(snapshot)
	}
}
