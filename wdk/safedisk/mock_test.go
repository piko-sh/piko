// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package safedisk

import (
	"errors"
	"io"
	"io/fs"
	"testing"
)

func TestMockFileHandle_ReadWrite(t *testing.T) {
	t.Parallel()

	file := NewMockFileHandle("test.txt", "/sandbox/test.txt", []byte("hello"))

	if got := file.Name(); got != "test.txt" {
		t.Errorf("Name() = %q, want %q", got, "test.txt")
	}
	if got := file.AbsolutePath(); got != "/sandbox/test.txt" {
		t.Errorf("AbsolutePath() = %q, want %q", got, "/sandbox/test.txt")
	}

	buffer := make([]byte, 10)
	n, err := file.Read(buffer)
	if err != nil {
		t.Errorf("Read() error = %v", err)
	}
	if got := string(buffer[:n]); got != "hello" {
		t.Errorf("Read() = %q, want %q", got, "hello")
	}

	_, _ = file.Seek(0, io.SeekEnd)
	_, err = file.Write([]byte(" world"))
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}
	if got := string(file.Data()); got != "hello world" {
		t.Errorf("Data() = %q, want %q", got, "hello world")
	}

	_, err = file.WriteString("!")
	if err != nil {
		t.Errorf("WriteString() error = %v", err)
	}
	if got := string(file.Data()); got != "hello world!" {
		t.Errorf("Data() = %q, want %q", got, "hello world!")
	}
}

func TestMockFileHandle_ReadAt(t *testing.T) {
	t.Parallel()

	file := NewMockFileHandle("test.txt", "/sandbox/test.txt", []byte("hello world"))

	buffer := make([]byte, 5)
	n, err := file.ReadAt(buffer, 6)
	if err != nil {
		t.Errorf("ReadAt() error = %v", err)
	}
	if got := string(buffer[:n]); got != "world" {
		t.Errorf("ReadAt() = %q, want %q", got, "world")
	}
}

func TestMockFileHandle_WriteAt(t *testing.T) {
	t.Parallel()

	file := NewMockFileHandle("test.txt", "/sandbox/test.txt", []byte("hello world"))

	_, err := file.WriteAt([]byte("WORLD"), 6)
	if err != nil {
		t.Errorf("WriteAt() error = %v", err)
	}
	if got := string(file.Data()); got != "hello WORLD" {
		t.Errorf("Data() = %q, want %q", got, "hello WORLD")
	}
}

func TestMockFileHandle_Seek(t *testing.T) {
	t.Parallel()

	file := NewMockFileHandle("test.txt", "/sandbox/test.txt", []byte("hello"))

	position, err := file.Seek(2, io.SeekStart)
	if err != nil || position != 2 {
		t.Errorf("Seek(2, SeekStart) = %d, %v; want 2, nil", position, err)
	}

	position, err = file.Seek(1, io.SeekCurrent)
	if err != nil || position != 3 {
		t.Errorf("Seek(1, SeekCurrent) = %d, %v; want 3, nil", position, err)
	}

	position, err = file.Seek(-2, io.SeekEnd)
	if err != nil || position != 3 {
		t.Errorf("Seek(-2, SeekEnd) = %d, %v; want 3, nil", position, err)
	}
}

func TestMockFileHandle_Truncate(t *testing.T) {
	t.Parallel()

	file := NewMockFileHandle("test.txt", "/sandbox/test.txt", []byte("hello world"))

	err := file.Truncate(5)
	if err != nil {
		t.Errorf("Truncate() error = %v", err)
	}
	if got := string(file.Data()); got != "hello" {
		t.Errorf("Data() = %q, want %q", got, "hello")
	}

	err = file.Truncate(10)
	if err != nil {
		t.Errorf("Truncate() error = %v", err)
	}
	if got := len(file.Data()); got != 10 {
		t.Errorf("len(Data()) = %d, want 10", got)
	}
}

func TestMockFileHandle_Stat(t *testing.T) {
	t.Parallel()

	file := NewMockFileHandle("test.txt", "/sandbox/test.txt", []byte("hello"))

	info, err := file.Stat()
	if err != nil {
		t.Errorf("Stat() error = %v", err)
	}
	if info.Name() != "test.txt" {
		t.Errorf("Stat().Name() = %q, want %q", info.Name(), "test.txt")
	}
	if info.Size() != 5 {
		t.Errorf("Stat().Size() = %d, want 5", info.Size())
	}
}

func TestMockFileHandle_ErrorInjection(t *testing.T) {
	t.Parallel()

	testErr := errors.New("injected error")
	file := NewMockFileHandle("test.txt", "/sandbox/test.txt", []byte("hello"))

	file.ReadErr = testErr
	_, err := file.Read(make([]byte, 10))
	if !errors.Is(err, testErr) {
		t.Errorf("Read() error = %v, want %v", err, testErr)
	}
	file.ReadErr = nil

	file.WriteErr = testErr
	_, err = file.Write([]byte("test"))
	if !errors.Is(err, testErr) {
		t.Errorf("Write() error = %v, want %v", err, testErr)
	}
	file.WriteErr = nil

	file.SyncErr = testErr
	err = file.Sync()
	if !errors.Is(err, testErr) {
		t.Errorf("Sync() error = %v, want %v", err, testErr)
	}
	file.SyncErr = nil

	file.CloseErr = testErr
	err = file.Close()
	if !errors.Is(err, testErr) {
		t.Errorf("Close() error = %v, want %v", err, testErr)
	}
}

func TestMockFileHandle_CloseState(t *testing.T) {
	t.Parallel()

	file := NewMockFileHandle("test.txt", "/sandbox/test.txt", nil)

	if file.IsClosed() {
		t.Error("IsClosed() = true before Close()")
	}

	_ = file.Close()

	if !file.IsClosed() {
		t.Error("IsClosed() = false after Close()")
	}
}

func TestMockSandbox_BasicOperations(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	if got := sandbox.Root(); got != "/sandbox" {
		t.Errorf("Root() = %q, want %q", got, "/sandbox")
	}
	if got := sandbox.Mode(); got != ModeReadWrite {
		t.Errorf("Mode() = %v, want %v", got, ModeReadWrite)
	}
	if sandbox.IsReadOnly() {
		t.Error("IsReadOnly() = true, want false")
	}
}

func TestMockSandbox_FileOperations(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	file, err := sandbox.Create("test.txt")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	_, _ = file.Write([]byte("hello"))
	_ = file.Close()

	data, err := sandbox.ReadFile("test.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("ReadFile() = %q, want %q", string(data), "hello")
	}

	info, err := sandbox.Stat("test.txt")
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Size() != 5 {
		t.Errorf("Stat().Size() = %d, want 5", info.Size())
	}
}

func TestMockSandbox_WriteFile(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	err := sandbox.WriteFile("test.txt", []byte("hello"), 0644)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	data, err := sandbox.ReadFile("test.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("ReadFile() = %q, want %q", string(data), "hello")
	}
}

func TestMockSandbox_Rename(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	sandbox.AddFile("old.txt", []byte("content"))

	err := sandbox.Rename("old.txt", "new.txt")
	if err != nil {
		t.Fatalf("Rename() error = %v", err)
	}

	_, err = sandbox.ReadFile("old.txt")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("ReadFile(old.txt) error = %v, want fs.ErrNotExist", err)
	}

	data, err := sandbox.ReadFile("new.txt")
	if err != nil {
		t.Fatalf("ReadFile(new.txt) error = %v", err)
	}
	if string(data) != "content" {
		t.Errorf("ReadFile() = %q, want %q", string(data), "content")
	}
}

func TestMockSandbox_Remove(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	sandbox.AddFile("test.txt", []byte("content"))

	err := sandbox.Remove("test.txt")
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	_, err = sandbox.ReadFile("test.txt")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("ReadFile() error = %v, want fs.ErrNotExist", err)
	}
}

func TestMockSandbox_ReadOnlyMode(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadOnly)
	defer func() { _ = sandbox.Close() }()
	sandbox.AddFile("test.txt", []byte("content"))

	_, err := sandbox.ReadFile("test.txt")
	if err != nil {
		t.Errorf("ReadFile() error = %v", err)
	}

	_, err = sandbox.Create("new.txt")
	if !errors.Is(err, errReadOnly) {
		t.Errorf("Create() error = %v, want errReadOnly", err)
	}

	err = sandbox.WriteFile("test.txt", []byte("new"), 0644)
	if !errors.Is(err, errReadOnly) {
		t.Errorf("WriteFile() error = %v, want errReadOnly", err)
	}

	err = sandbox.Remove("test.txt")
	if !errors.Is(err, errReadOnly) {
		t.Errorf("Remove() error = %v, want errReadOnly", err)
	}
}

func TestMockSandbox_ErrorInjection(t *testing.T) {
	t.Parallel()

	testErr := errors.New("injected error")
	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	sandbox.AddFile("test.txt", []byte("content"))

	sandbox.CreateErr = testErr
	_, err := sandbox.Create("new.txt")
	if !errors.Is(err, testErr) {
		t.Errorf("Create() error = %v, want %v", err, testErr)
	}
	sandbox.CreateErr = nil

	sandbox.ReadFileErr = testErr
	_, err = sandbox.ReadFile("test.txt")
	if !errors.Is(err, testErr) {
		t.Errorf("ReadFile() error = %v, want %v", err, testErr)
	}
	sandbox.ReadFileErr = nil

	sandbox.StatErr = testErr
	_, err = sandbox.Stat("test.txt")
	if !errors.Is(err, testErr) {
		t.Errorf("Stat() error = %v, want %v", err, testErr)
	}
	sandbox.StatErr = nil

	sandbox.WriteFileErr = testErr
	err = sandbox.WriteFile("test.txt", []byte("new"), 0644)
	if !errors.Is(err, testErr) {
		t.Errorf("WriteFile() error = %v, want %v", err, testErr)
	}
}

func TestMockSandbox_CallCounts(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	sandbox.AddFile("test.txt", []byte("content"))

	_, _ = sandbox.Open("test.txt")
	_, _ = sandbox.Open("test.txt")
	_, _ = sandbox.ReadFile("test.txt")

	if got := sandbox.CallCounts["Open"]; got != 2 {
		t.Errorf("CallCounts[Open] = %d, want 2", got)
	}
	if got := sandbox.CallCounts["ReadFile"]; got != 1 {
		t.Errorf("CallCounts[ReadFile] = %d, want 1", got)
	}
}

func TestMockSandbox_CreateTemp(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	file, err := sandbox.CreateTemp("tmp", "test-*.txt")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}

	if file.Name() == "" {
		t.Error("CreateTemp() returned file with empty name")
	}

	_, _ = file.Write([]byte("temp content"))
	_ = file.Close()
}

func TestMockSandbox_OpenFile(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	file, err := sandbox.OpenFile("new.txt", 0x40, 0644)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	_ = file.Close()

	_, err = sandbox.Stat("new.txt")
	if err != nil {
		t.Errorf("Stat() error = %v after OpenFile with O_CREATE", err)
	}

	_, err = sandbox.OpenFile("missing.txt", 0, 0644)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("OpenFile() error = %v, want fs.ErrNotExist", err)
	}
}

func TestMockFileHandle_Chmod(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		file := NewMockFileHandle("test.txt", "/sandbox/test.txt", nil)
		if err := file.Chmod(0600); err != nil {
			t.Errorf("Chmod() error = %v, want nil", err)
		}
	})

	t.Run("error injection", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("chmod error")
		file := NewMockFileHandle("test.txt", "/sandbox/test.txt", nil)
		file.ChmodErr = testErr
		if err := file.Chmod(0600); !errors.Is(err, testErr) {
			t.Errorf("Chmod() error = %v, want %v", err, testErr)
		}
	})
}

func TestMockFileHandle_ReadDir(t *testing.T) {
	t.Parallel()

	t.Run("with entries", func(t *testing.T) {
		t.Parallel()
		file := NewMockFileHandle("dir", "/sandbox/directory", nil)
		file.DirEntries = []fs.DirEntry{
			&mockDirEntry{name: "a.txt", isDir: false, info: &mockFileInfo{name: "a.txt", size: 10}},
			&mockDirEntry{name: "b.txt", isDir: false, info: &mockFileInfo{name: "b.txt", size: 20}},
			&mockDirEntry{name: "sub", isDir: true, info: &mockFileInfo{name: "sub", isDir: true}},
		}

		entries, err := file.ReadDir(-1)
		if err != nil {
			t.Fatalf("ReadDir(-1) error = %v", err)
		}
		if len(entries) != 3 {
			t.Errorf("ReadDir(-1) returned %d entries, want 3", len(entries))
		}
	})

	t.Run("with limit", func(t *testing.T) {
		t.Parallel()
		file := NewMockFileHandle("dir", "/sandbox/directory", nil)
		file.DirEntries = []fs.DirEntry{
			&mockDirEntry{name: "a.txt", isDir: false, info: &mockFileInfo{name: "a.txt"}},
			&mockDirEntry{name: "b.txt", isDir: false, info: &mockFileInfo{name: "b.txt"}},
			&mockDirEntry{name: "c.txt", isDir: false, info: &mockFileInfo{name: "c.txt"}},
		}

		entries, err := file.ReadDir(2)
		if err != nil {
			t.Fatalf("ReadDir(2) error = %v", err)
		}
		if len(entries) != 2 {
			t.Errorf("ReadDir(2) returned %d entries, want 2", len(entries))
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		t.Parallel()
		file := NewMockFileHandle("dir", "/sandbox/directory", nil)

		entries, err := file.ReadDir(-1)
		if err != nil {
			t.Fatalf("ReadDir(-1) error = %v", err)
		}
		if entries != nil {
			t.Errorf("ReadDir(-1) = %v, want nil", entries)
		}
	})

	t.Run("error injection", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("readdir error")
		file := NewMockFileHandle("dir", "/sandbox/directory", nil)
		file.ReadDirErr = testErr

		_, err := file.ReadDir(-1)
		if !errors.Is(err, testErr) {
			t.Errorf("ReadDir() error = %v, want %v", err, testErr)
		}
	})
}

func TestMockFileHandle_Fd(t *testing.T) {
	t.Parallel()

	file := NewMockFileHandle("test.txt", "/sandbox/test.txt", nil)
	if got := file.Fd(); got != 0 {
		t.Errorf("Fd() = %d, want 0", got)
	}
}

func TestMockFileHandle_StatInfo(t *testing.T) {
	t.Parallel()

	t.Run("custom stat info", func(t *testing.T) {
		t.Parallel()
		file := NewMockFileHandle("test.txt", "/sandbox/test.txt", []byte("data"))
		customInfo := &mockFileInfo{name: "custom.txt", size: 999, isDir: true}
		file.StatInfo = customInfo

		info, err := file.Stat()
		if err != nil {
			t.Fatalf("Stat() error = %v", err)
		}
		if info.Name() != "custom.txt" {
			t.Errorf("Stat().Name() = %q, want %q", info.Name(), "custom.txt")
		}
		if info.Size() != 999 {
			t.Errorf("Stat().Size() = %d, want 999", info.Size())
		}
		if !info.IsDir() {
			t.Error("Stat().IsDir() = false, want true")
		}
	})

	t.Run("stat error", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("stat error")
		file := NewMockFileHandle("test.txt", "/sandbox/test.txt", nil)
		file.StatErr = testErr

		_, err := file.Stat()
		if !errors.Is(err, testErr) {
			t.Errorf("Stat() error = %v, want %v", err, testErr)
		}
	})
}

func TestMockFileHandle_ReadAtBeyondEnd(t *testing.T) {
	t.Parallel()

	file := NewMockFileHandle("test.txt", "/sandbox/test.txt", []byte("hello"))

	buffer := make([]byte, 5)
	n, err := file.ReadAt(buffer, 100)
	if err != nil {
		t.Errorf("ReadAt() error = %v", err)
	}
	if n != 0 {
		t.Errorf("ReadAt() n = %d, want 0", n)
	}
}

func TestMockFileHandle_ReadAtError(t *testing.T) {
	t.Parallel()

	testErr := errors.New("readat error")
	file := NewMockFileHandle("test.txt", "/sandbox/test.txt", []byte("hello"))
	file.ReadAtErr = testErr

	_, err := file.ReadAt(make([]byte, 5), 0)
	if !errors.Is(err, testErr) {
		t.Errorf("ReadAt() error = %v, want %v", err, testErr)
	}
}

func TestMockFileHandle_WriteAtError(t *testing.T) {
	t.Parallel()

	testErr := errors.New("writeat error")
	file := NewMockFileHandle("test.txt", "/sandbox/test.txt", []byte("hello"))
	file.WriteAtErr = testErr

	_, err := file.WriteAt([]byte("X"), 0)
	if !errors.Is(err, testErr) {
		t.Errorf("WriteAt() error = %v, want %v", err, testErr)
	}
}

func TestMockFileHandle_SeekError(t *testing.T) {
	t.Parallel()

	testErr := errors.New("seek error")
	file := NewMockFileHandle("test.txt", "/sandbox/test.txt", []byte("hello"))
	file.SeekErr = testErr

	_, err := file.Seek(0, io.SeekStart)
	if !errors.Is(err, testErr) {
		t.Errorf("Seek() error = %v, want %v", err, testErr)
	}
}

func TestMockFileHandle_TruncateError(t *testing.T) {
	t.Parallel()

	testErr := errors.New("truncate error")
	file := NewMockFileHandle("test.txt", "/sandbox/test.txt", []byte("hello"))
	file.TruncateErr = testErr

	err := file.Truncate(0)
	if !errors.Is(err, testErr) {
		t.Errorf("Truncate() error = %v, want %v", err, testErr)
	}
}

func TestMockFileHandle_SeekNegativeClamp(t *testing.T) {
	t.Parallel()

	file := NewMockFileHandle("test.txt", "/sandbox/test.txt", []byte("hello"))
	position, err := file.Seek(-100, io.SeekStart)
	if err != nil {
		t.Errorf("Seek() error = %v", err)
	}
	if position != 0 {
		t.Errorf("Seek(-100, SeekStart) = %d, want 0 (clamped)", position)
	}
}

func TestMockFileHandle_ReadEOF(t *testing.T) {
	t.Parallel()

	file := NewMockFileHandle("test.txt", "/sandbox/test.txt", []byte("hi"))

	buffer := make([]byte, 10)
	n, _ := file.Read(buffer)
	if n != 2 {
		t.Fatalf("first Read() = %d, want 2", n)
	}

	_, err := file.Read(buffer)
	if !errors.Is(err, io.EOF) {
		t.Errorf("second Read() error = %v, want io.EOF", err)
	}
}

func TestMockSandbox_GetFile(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	sandbox.AddFile("existing.txt", []byte("data"))

	t.Run("existing file", func(t *testing.T) {
		t.Parallel()
		file := sandbox.GetFile("existing.txt")
		if file == nil {
			t.Fatal("GetFile() returned nil for existing file")
		}
		if string(file.Data()) != "data" {
			t.Errorf("GetFile().Data() = %q, want %q", string(file.Data()), "data")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		t.Parallel()
		file := sandbox.GetFile("missing.txt")
		if file != nil {
			t.Errorf("GetFile() = %v, want nil for missing file", file)
		}
	})
}

func TestMockSandbox_Lstat(t *testing.T) {
	t.Parallel()

	t.Run("file exists", func(t *testing.T) {
		t.Parallel()
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("test.txt", []byte("content"))

		info, err := sandbox.Lstat("test.txt")
		if err != nil {
			t.Fatalf("Lstat() error = %v", err)
		}
		if info.Name() != "test.txt" {
			t.Errorf("Lstat().Name() = %q, want %q", info.Name(), "test.txt")
		}
	})

	t.Run("file missing", func(t *testing.T) {
		t.Parallel()
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		_, err := sandbox.Lstat("missing.txt")
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("Lstat() error = %v, want fs.ErrNotExist", err)
		}
	})

	t.Run("error injection", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("lstat error")
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("test.txt", []byte("content"))
		sandbox.LstatErr = testErr

		_, err := sandbox.Lstat("test.txt")
		if !errors.Is(err, testErr) {
			t.Errorf("Lstat() error = %v, want %v", err, testErr)
		}
	})

	t.Run("call count tracked", func(t *testing.T) {
		t.Parallel()
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("test.txt", []byte("content"))

		_, _ = sandbox.Lstat("test.txt")
		_, _ = sandbox.Lstat("test.txt")

		if got := sandbox.CallCounts["Lstat"]; got != 2 {
			t.Errorf("CallCounts[Lstat] = %d, want 2", got)
		}
	})
}

func TestMockSandbox_Mkdir(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		if err := sandbox.Mkdir("subdir", 0755); err != nil {
			t.Errorf("Mkdir() error = %v", err)
		}
	})

	t.Run("read-only mode", func(t *testing.T) {
		t.Parallel()
		sandbox := NewMockSandbox("/sandbox", ModeReadOnly)
		defer func() { _ = sandbox.Close() }()
		if err := sandbox.Mkdir("subdir", 0755); !errors.Is(err, errReadOnly) {
			t.Errorf("Mkdir() error = %v, want errReadOnly", err)
		}
	})

	t.Run("error injection", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("mkdir error")
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.MkdirErr = testErr
		if err := sandbox.Mkdir("subdir", 0755); !errors.Is(err, testErr) {
			t.Errorf("Mkdir() error = %v, want %v", err, testErr)
		}
	})
}

func TestMockSandbox_MkdirAll(t *testing.T) {
	t.Parallel()

	t.Run("success with nested path", func(t *testing.T) {
		t.Parallel()
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		if err := sandbox.MkdirAll("a/b/c", 0755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}

		for _, path := range []string{"a", "a/b", "a/b/c"} {
			info, err := sandbox.Stat(path)
			if err != nil {
				t.Errorf("Stat(%q) error = %v", path, err)
				continue
			}
			if !info.IsDir() {
				t.Errorf("Stat(%q).IsDir() = false, want true", path)
			}
		}
	})

	t.Run("read-only mode", func(t *testing.T) {
		t.Parallel()
		sandbox := NewMockSandbox("/sandbox", ModeReadOnly)
		defer func() { _ = sandbox.Close() }()
		if err := sandbox.MkdirAll("a/b", 0755); !errors.Is(err, errReadOnly) {
			t.Errorf("MkdirAll() error = %v, want errReadOnly", err)
		}
	})

	t.Run("error injection", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("mkdirall error")
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.MkdirAllErr = testErr
		if err := sandbox.MkdirAll("a/b", 0755); !errors.Is(err, testErr) {
			t.Errorf("MkdirAll() error = %v, want %v", err, testErr)
		}
	})

	t.Run("dot path", func(t *testing.T) {
		t.Parallel()
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		if err := sandbox.MkdirAll(".", 0755); err != nil {
			t.Errorf("MkdirAll(\".\") error = %v", err)
		}
	})

	t.Run("empty path", func(t *testing.T) {
		t.Parallel()
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		if err := sandbox.MkdirAll("", 0755); err != nil {
			t.Errorf("MkdirAll(\"\") error = %v", err)
		}
	})
}

func TestMockSandbox_RemoveAll(t *testing.T) {
	t.Parallel()

	t.Run("removes file and children", func(t *testing.T) {
		t.Parallel()
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("dir/a.txt", []byte("a"))
		sandbox.AddFile("dir/sub/b.txt", []byte("b"))
		sandbox.AddFile("other.txt", []byte("other"))

		if err := sandbox.RemoveAll("dir"); err != nil {
			t.Fatalf("RemoveAll() error = %v", err)
		}

		if f := sandbox.GetFile("dir/a.txt"); f != nil {
			t.Error("dir/a.txt still exists after RemoveAll")
		}
		if f := sandbox.GetFile("dir/sub/b.txt"); f != nil {
			t.Error("dir/sub/b.txt still exists after RemoveAll")
		}
		if f := sandbox.GetFile("other.txt"); f == nil {
			t.Error("other.txt was removed unexpectedly")
		}
	})

	t.Run("read-only mode", func(t *testing.T) {
		t.Parallel()
		sandbox := NewMockSandbox("/sandbox", ModeReadOnly)
		defer func() { _ = sandbox.Close() }()
		if err := sandbox.RemoveAll("dir"); !errors.Is(err, errReadOnly) {
			t.Errorf("RemoveAll() error = %v, want errReadOnly", err)
		}
	})

	t.Run("error injection", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("removeall error")
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.RemoveAllErr = testErr
		if err := sandbox.RemoveAll("dir"); !errors.Is(err, testErr) {
			t.Errorf("RemoveAll() error = %v, want %v", err, testErr)
		}
	})
}

func TestMockSandbox_Chmod(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		if err := sandbox.Chmod("file.txt", 0600); err != nil {
			t.Errorf("Chmod() error = %v", err)
		}
	})

	t.Run("read-only mode", func(t *testing.T) {
		t.Parallel()
		sandbox := NewMockSandbox("/sandbox", ModeReadOnly)
		defer func() { _ = sandbox.Close() }()
		if err := sandbox.Chmod("file.txt", 0600); !errors.Is(err, errReadOnly) {
			t.Errorf("Chmod() error = %v, want errReadOnly", err)
		}
	})

	t.Run("error injection", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("chmod error")
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.ChmodErr = testErr
		if err := sandbox.Chmod("file.txt", 0600); !errors.Is(err, testErr) {
			t.Errorf("Chmod() error = %v, want %v", err, testErr)
		}
	})

	t.Run("with ChmodFunc", func(t *testing.T) {
		t.Parallel()
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		var calledName string
		var calledMode fs.FileMode
		sandbox.ChmodFunc = func(name string, mode fs.FileMode) error {
			calledName = name
			calledMode = mode
			return nil
		}

		if err := sandbox.Chmod("test.txt", 0755); err != nil {
			t.Errorf("Chmod() error = %v", err)
		}
		if calledName != "test.txt" {
			t.Errorf("ChmodFunc called with name = %q, want %q", calledName, "test.txt")
		}
		if calledMode != 0755 {
			t.Errorf("ChmodFunc called with mode = %o, want 0755", calledMode)
		}
	})
}

func TestMockSandbox_MkdirTemp(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		path, err := sandbox.MkdirTemp("tmp", "test-*")
		if err != nil {
			t.Fatalf("MkdirTemp() error = %v", err)
		}
		if path == "" {
			t.Error("MkdirTemp() returned empty path")
		}
		if got := path; got != "tmp/temp-test-*-12345" {
			t.Errorf("MkdirTemp() = %q, want %q", got, "tmp/temp-test-*-12345")
		}
	})

	t.Run("error injection", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("mkdirtemp error")
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.MkdirTempErr = testErr
		_, err := sandbox.MkdirTemp("tmp", "test-*")
		if !errors.Is(err, testErr) {
			t.Errorf("MkdirTemp() error = %v, want %v", err, testErr)
		}
	})

	t.Run("read-only mode", func(t *testing.T) {
		t.Parallel()
		sandbox := NewMockSandbox("/sandbox", ModeReadOnly)
		defer func() { _ = sandbox.Close() }()
		_, err := sandbox.MkdirTemp("tmp", "test-*")
		if !errors.Is(err, errReadOnly) {
			t.Errorf("MkdirTemp() error = %v, want errReadOnly", err)
		}
	})
}

func TestMockSandbox_Close(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		if err := sandbox.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	t.Run("error injection", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("close error")
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		sandbox.CloseErr = testErr
		if err := sandbox.Close(); !errors.Is(err, testErr) {
			t.Errorf("Close() error = %v, want %v", err, testErr)
		}
	})
}

func TestMockSandbox_RelPath(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "relative path", input: "file.txt", expected: "file.txt"},
		{name: "nested relative", input: "a/b/c.txt", expected: "a/b/c.txt"},
		{name: "absolute path", input: "/somewhere/else", expected: "/somewhere/else"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := sandbox.RelPath(tc.input); got != tc.expected {
				t.Errorf("RelPath(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestMockSandbox_WriteFileAtomic(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		if err := sandbox.WriteFileAtomic("test.txt", []byte("atomic"), 0644); err != nil {
			t.Fatalf("WriteFileAtomic() error = %v", err)
		}

		data, err := sandbox.ReadFile("test.txt")
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}
		if string(data) != "atomic" {
			t.Errorf("ReadFile() = %q, want %q", string(data), "atomic")
		}
	})

	t.Run("error injection", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("writefileatomic error")
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.WriteFileAtomicErr = testErr
		if err := sandbox.WriteFileAtomic("test.txt", []byte("data"), 0644); !errors.Is(err, testErr) {
			t.Errorf("WriteFileAtomic() error = %v, want %v", err, testErr)
		}
	})

	t.Run("read-only mode", func(t *testing.T) {
		t.Parallel()
		sandbox := NewMockSandbox("/sandbox", ModeReadOnly)
		defer func() { _ = sandbox.Close() }()
		if err := sandbox.WriteFileAtomic("test.txt", []byte("data"), 0644); !errors.Is(err, errReadOnly) {
			t.Errorf("WriteFileAtomic() error = %v, want errReadOnly", err)
		}
	})
}

func TestMockSandbox_ReadDir(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		entries, err := sandbox.ReadDir(".")
		if err != nil {
			t.Errorf("ReadDir() error = %v", err)
		}
		if entries != nil {
			t.Errorf("ReadDir() = %v, want nil", entries)
		}
	})

	t.Run("error injection", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("readdir error")
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.ReadDirErr = testErr
		_, err := sandbox.ReadDir(".")
		if !errors.Is(err, testErr) {
			t.Errorf("ReadDir() error = %v, want %v", err, testErr)
		}
	})
}

func TestMockSandbox_OpenErr(t *testing.T) {
	t.Parallel()

	testErr := errors.New("open error")
	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	sandbox.OpenErr = testErr

	_, err := sandbox.Open("test.txt")
	if !errors.Is(err, testErr) {
		t.Errorf("Open() error = %v, want %v", err, testErr)
	}
}

func TestMockSandbox_OpenFileErr(t *testing.T) {
	t.Parallel()

	testErr := errors.New("openfile error")
	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	sandbox.OpenFileErr = testErr

	_, err := sandbox.OpenFile("test.txt", 0x40, 0644)
	if !errors.Is(err, testErr) {
		t.Errorf("OpenFile() error = %v, want %v", err, testErr)
	}
}

func TestMockSandbox_RemoveErr(t *testing.T) {
	t.Parallel()

	testErr := errors.New("remove error")
	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	sandbox.RemoveErr = testErr

	err := sandbox.Remove("test.txt")
	if !errors.Is(err, testErr) {
		t.Errorf("Remove() error = %v, want %v", err, testErr)
	}
}

func TestMockSandbox_RenameErr(t *testing.T) {
	t.Parallel()

	testErr := errors.New("rename error")
	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	sandbox.RenameErr = testErr

	err := sandbox.Rename("old.txt", "new.txt")
	if !errors.Is(err, testErr) {
		t.Errorf("Rename() error = %v, want %v", err, testErr)
	}
}

func TestMockSandbox_CreateTempErr(t *testing.T) {
	t.Parallel()

	testErr := errors.New("createtemp error")
	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	sandbox.CreateTempErr = testErr

	_, err := sandbox.CreateTemp(".", "test-*")
	if !errors.Is(err, testErr) {
		t.Errorf("CreateTemp() error = %v, want %v", err, testErr)
	}
}

func TestMockSandbox_CreateTempReadOnly(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadOnly)
	defer func() { _ = sandbox.Close() }()
	_, err := sandbox.CreateTemp(".", "test-*")
	if !errors.Is(err, errReadOnly) {
		t.Errorf("CreateTemp() error = %v, want errReadOnly", err)
	}
}

func TestMockSandbox_CreateTemp_NextTempFileErrors(t *testing.T) {
	t.Parallel()

	t.Run("next temp file write error", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("temp write error")
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.NextTempFileWriteErr = testErr

		file, err := sandbox.CreateTemp(".", "test-*")
		if err != nil {
			t.Fatalf("CreateTemp() error = %v", err)
		}

		_, err = file.Write([]byte("data"))
		if !errors.Is(err, testErr) {
			t.Errorf("Write() error = %v, want %v", err, testErr)
		}
	})

	t.Run("next temp file sync error", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("temp sync error")
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.NextTempFileSyncErr = testErr

		file, err := sandbox.CreateTemp(".", "test-*")
		if err != nil {
			t.Fatalf("CreateTemp() error = %v", err)
		}

		err = file.Sync()
		if !errors.Is(err, testErr) {
			t.Errorf("Sync() error = %v, want %v", err, testErr)
		}
	})

	t.Run("next temp file close error", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("temp close error")
		sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.NextTempFileCloseErr = testErr

		file, err := sandbox.CreateTemp(".", "test-*")
		if err != nil {
			t.Fatalf("CreateTemp() error = %v", err)
		}

		err = file.Close()
		if !errors.Is(err, testErr) {
			t.Errorf("Close() error = %v, want %v", err, testErr)
		}
	})
}

func TestMockSandbox_OpenFile_ReadOnlyWriteFlags(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadOnly)
	defer func() { _ = sandbox.Close() }()

	_, err := sandbox.OpenFile("test.txt", flagWrite, 0644)
	if !errors.Is(err, errReadOnly) {
		t.Errorf("OpenFile(flagWrite) error = %v, want errReadOnly", err)
	}

	_, err = sandbox.OpenFile("test.txt", flagReadWrite, 0644)
	if !errors.Is(err, errReadOnly) {
		t.Errorf("OpenFile(flagReadWrite) error = %v, want errReadOnly", err)
	}

	_, err = sandbox.OpenFile("test.txt", flagAppend, 0644)
	if !errors.Is(err, errReadOnly) {
		t.Errorf("OpenFile(flagAppend) error = %v, want errReadOnly", err)
	}

	_, err = sandbox.OpenFile("test.txt", flagTrunc, 0644)
	if !errors.Is(err, errReadOnly) {
		t.Errorf("OpenFile(flagTrunc) error = %v, want errReadOnly", err)
	}
}

func TestMockSandbox_Rename_ReadOnly(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadOnly)
	defer func() { _ = sandbox.Close() }()
	sandbox.AddFile("old.txt", []byte("data"))

	err := sandbox.Rename("old.txt", "new.txt")
	if !errors.Is(err, errReadOnly) {
		t.Errorf("Rename() error = %v, want errReadOnly", err)
	}
}

func TestMockSandbox_Rename_NotExist(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	err := sandbox.Rename("missing.txt", "new.txt")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Rename() error = %v, want fs.ErrNotExist", err)
	}
}

func TestMockSandbox_WalkDir_SingleFile(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	sandbox.AddFile("file.txt", []byte("content"))

	var visited []string
	err := sandbox.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		visited = append(visited, path)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir() error = %v", err)
	}

	if len(visited) != 1 {
		t.Errorf("WalkDir visited %d paths, want 1: %v", len(visited), visited)
	}
	if len(visited) > 0 && visited[0] != "file.txt" {
		t.Errorf("WalkDir visited %v, want [file.txt]", visited)
	}
}

func TestMockSandbox_WalkDir_DirectoryTree(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	sandbox.AddFile("a/b/c.txt", []byte("c"))
	sandbox.AddFile("a/d.txt", []byte("d"))

	var visited []string
	err := sandbox.WalkDir("a", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		visited = append(visited, path)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir() error = %v", err)
	}

	if len(visited) != 2 {
		t.Errorf("WalkDir visited %d paths, want 2: %v", len(visited), visited)
	}
}

func TestMockSandbox_WalkDir_SkipDir(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	dirFile := sandbox.AddFile("a", nil)
	dirFile.StatInfo = &mockFileInfo{name: "a", isDir: true}

	sandbox.AddFile("a/skip.txt", []byte("skip"))
	sandbox.AddFile("b.txt", []byte("b"))

	var visited []string
	err := sandbox.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		visited = append(visited, path)
		if d.IsDir() && path == "a" {
			return fs.SkipDir
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir() error = %v", err)
	}

	for _, v := range visited {
		if v == "a/skip.txt" {
			t.Error("WalkDir should have skipped a/skip.txt but visited it")
		}
	}
}

func TestMockSandbox_WalkDir_SkipAll(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	sandbox.AddFile("a.txt", []byte("a"))
	sandbox.AddFile("b.txt", []byte("b"))
	sandbox.AddFile("c.txt", []byte("c"))

	var visited []string
	err := sandbox.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		visited = append(visited, path)
		return fs.SkipAll
	})
	if err != nil {
		t.Fatalf("WalkDir() error = %v", err)
	}

	if len(visited) != 1 {
		t.Errorf("WalkDir visited %d paths, want 1 (stopped by SkipAll): %v", len(visited), visited)
	}
}

func TestMockSandbox_WalkDir_Error(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	sandbox.AddFile("file.txt", []byte("data"))

	customErr := errors.New("walk error")
	err := sandbox.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		return customErr
	})
	if !errors.Is(err, customErr) {
		t.Errorf("WalkDir() error = %v, want %v", err, customErr)
	}
}

func TestMockSandbox_WalkDir_WalkDirErr(t *testing.T) {
	t.Parallel()

	testErr := errors.New("walkdir injected error")
	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	sandbox.WalkDirErr = testErr

	err := sandbox.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		return nil
	})
	if !errors.Is(err, testErr) {
		t.Errorf("WalkDir() error = %v, want %v", err, testErr)
	}
}

func TestMockSandbox_WalkDir_EmptyRoot(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	var visited []string
	err := sandbox.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		visited = append(visited, path)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir() error = %v", err)
	}

	if len(visited) != 0 {
		t.Errorf("WalkDir visited %d paths, want 0: %v", len(visited), visited)
	}
}

func TestMockSandbox_WalkDir_StatError(t *testing.T) {
	t.Parallel()

	sandbox := NewMockSandbox("/sandbox", ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	statErr := errors.New("stat error")
	file := sandbox.AddFile("bad.txt", []byte("data"))
	file.StatErr = statErr

	var gotErr error
	err := sandbox.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			gotErr = err
			return nil
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir() error = %v", err)
	}
	if !errors.Is(gotErr, statErr) {
		t.Errorf("WalkDir callback received error = %v, want %v", gotErr, statErr)
	}
}

func TestSplitPath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{name: "empty", input: "", expected: []string{"."}},
		{name: "dot", input: ".", expected: []string{"."}},
		{name: "single", input: "foo", expected: []string{"foo"}},
		{name: "nested", input: "a/b/c", expected: []string{"a", "b", "c"}},
		{name: "trailing slash", input: "a/b/", expected: []string{"a", "b"}},
		{name: "leading slash", input: "/a/b", expected: []string{"a", "b"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := splitPath(tc.input)
			if len(got) != len(tc.expected) {
				t.Fatalf("splitPath(%q) = %v (len %d), want %v (len %d)", tc.input, got, len(got), tc.expected, len(tc.expected))
			}
			for i := range got {
				if got[i] != tc.expected[i] {
					t.Errorf("splitPath(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.expected[i])
				}
			}
		})
	}
}

func TestMatchesRoot(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		path     string
		root     string
		expected bool
	}{
		{name: "empty root", path: "a/b", root: "", expected: true},
		{name: "dot root", path: "a/b", root: ".", expected: true},
		{name: "exact match", path: "a", root: "a", expected: true},
		{name: "child path", path: "a/b", root: "a", expected: true},
		{name: "no match", path: "b/c", root: "a", expected: false},
		{name: "prefix but not child", path: "abc", root: "a", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := matchesRoot(tc.path, tc.root); got != tc.expected {
				t.Errorf("matchesRoot(%q, %q) = %v, want %v", tc.path, tc.root, got, tc.expected)
			}
		})
	}
}

func TestIsPathSkipped(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		path         string
		skipPrefixes []string
		expected     bool
	}{
		{name: "no prefixes", path: "a/b", skipPrefixes: nil, expected: false},
		{name: "matching prefix", path: "a/b/c", skipPrefixes: []string{"a/b"}, expected: true},
		{name: "non-matching prefix", path: "x/y", skipPrefixes: []string{"a/b"}, expected: false},
		{name: "exact match is not skipped", path: "a/b", skipPrefixes: []string{"a/b"}, expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isPathSkipped(tc.path, tc.skipPrefixes); got != tc.expected {
				t.Errorf("isPathSkipped(%q, %v) = %v, want %v", tc.path, tc.skipPrefixes, got, tc.expected)
			}
		})
	}
}

func TestSortStrings(t *testing.T) {
	t.Parallel()

	input := []string{"c", "a", "b"}
	sortStrings(input)
	expected := []string{"a", "b", "c"}
	for i := range input {
		if input[i] != expected[i] {
			t.Errorf("sortStrings result[%d] = %q, want %q", i, input[i], expected[i])
		}
	}
}

func TestMockFileInfo_Methods(t *testing.T) {
	t.Parallel()

	info := &mockFileInfo{
		name:  "test.txt",
		size:  42,
		mode:  0644,
		isDir: false,
	}

	if info.Name() != "test.txt" {
		t.Errorf("Name() = %q, want %q", info.Name(), "test.txt")
	}
	if info.Size() != 42 {
		t.Errorf("Size() = %d, want 42", info.Size())
	}
	if info.Mode() != 0644 {
		t.Errorf("Mode() = %o, want 0644", info.Mode())
	}
	if info.IsDir() {
		t.Error("IsDir() = true, want false")
	}
	if info.Sys() != nil {
		t.Errorf("Sys() = %v, want nil", info.Sys())
	}
	_ = info.ModTime()
}

func TestMockDirEntry_Methods(t *testing.T) {
	t.Parallel()

	info := &mockFileInfo{name: "entry", mode: fs.ModeDir | 0755, isDir: true}
	entry := &mockDirEntry{name: "entry", isDir: true, info: info}

	if entry.Name() != "entry" {
		t.Errorf("Name() = %q, want %q", entry.Name(), "entry")
	}
	if !entry.IsDir() {
		t.Error("IsDir() = false, want true")
	}
	if entry.Type()&fs.ModeDir == 0 {
		t.Error("Type() does not include ModeDir")
	}
	gotInfo, err := entry.Info()
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	if gotInfo != info {
		t.Error("Info() returned different info")
	}
}
