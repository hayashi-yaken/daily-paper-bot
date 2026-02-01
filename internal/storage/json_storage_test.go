package storage

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTest はテスト用のJSONStorageと一時ファイルパスを準備します
func setupTest(t *testing.T) (*JSONStorage, string, func()) {
	t.Helper()

	// 一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "storage_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// テスト用のファイルパス
	testFilePath := filepath.Join(tmpDir, "posted.json")

	// テスト用のJSONStorageインスタンスを作成
	storage := &JSONStorage{
		data: &PostedData{Posted: make(map[string]PostedEntry)},
		path: testFilePath,
	}

	// クリーンアップ関数
	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return storage, testFilePath, cleanup
}

func TestLoad_FileNotExist(t *testing.T) {
	storage, _, cleanup := setupTest(t)
	defer cleanup()

	err := storage.load()
	if err != nil {
		t.Fatalf("load() failed for non-existent file: %v", err)
	}

	if storage.data == nil {
		t.Fatal("storage.data should be initialized, but got nil")
	}
	if len(storage.data.Posted) != 0 {
		t.Errorf("expected 0 posted entries, got %d", len(storage.data.Posted))
	}
}

func TestLoad_FileExists(t *testing.T) {
	storage, testFilePath, cleanup := setupTest(t)
	defer cleanup()

	content := `{"posted":{"paper1":{"date":"2023-01-01","venue":"ICLR 2023"}}}`
	if err := os.WriteFile(testFilePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	if err := storage.load(); err != nil {
		t.Fatalf("load() failed: %v", err)
	}

	if !storage.IsPosted("paper1") {
		t.Error("expected 'paper1' to be posted, but it was not")
	}
	if storage.IsPosted("paper2") {
		t.Error("expected 'paper2' not to be posted, but it was")
	}
}

func TestIsPosted(t *testing.T) {
	storage, _, cleanup := setupTest(t)
	defer cleanup()

	storage.data.Posted["paper1"] = PostedEntry{Date: "2023-01-01", Venue: "TestConf"}

	if !storage.IsPosted("paper1") {
		t.Error("expected 'paper1' to be posted, but it was not")
	}
	if storage.IsPosted("paper2") {
		t.Error("expected 'paper2' not to be posted, but it was")
	}
}

func TestAdd(t *testing.T) {
	storage, _, cleanup := setupTest(t)
	defer cleanup()

	storage.Add("paper1", "TestConf")

	if !storage.IsPosted("paper1") {
		t.Error("expected 'paper1' to be added, but it was not")
	}
	if _, ok := storage.data.Posted["paper1"]; !ok {
		t.Fatal("entry 'paper1' not found in map")
	}
}

func TestSaveAndLoad(t *testing.T) {
	storage, testFilePath, cleanup := setupTest(t)
	defer cleanup()

	// データを追加
	storage.Add("paper1", "ConfA")
	storage.Add("paper2", "ConfB")

	// 保存
	if err := storage.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// 別のインスタンスで読み込む
	storage2 := &JSONStorage{
		data: &PostedData{Posted: make(map[string]PostedEntry)},
		path: testFilePath,
	}
	if err := storage2.load(); err != nil {
		t.Fatalf("load() failed: %v", err)
	}

	// データが正しく読み込まれたか確認
	if !storage2.IsPosted("paper1") {
		t.Error("expected 'paper1' to be loaded, but it was not")
	}
	if !storage2.IsPosted("paper2") {
		t.Error("expected 'paper2' to be loaded, but it was not")
	}
	if storage2.IsPosted("paper3") {
		t.Error("expected 'paper3' not to be loaded, but it was")
	}
	if storage2.data.Posted["paper1"].Venue != "ConfA" {
		t.Errorf("expected venue 'ConfA', got '%s'", storage2.data.Posted["paper1"].Venue)
	}
}
