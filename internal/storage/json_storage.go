package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const storagePath = "data/posted.json"

type JSONStorage struct {
	data *PostedData
	path string
}

func NewJSONStorage() (*JSONStorage, error) {
	absPath, err := filepath.Abs(storagePath)
	if err != nil {
		return nil, err
	}

	s := &JSONStorage{path: absPath}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// load はJSONファイルからデータを読み込む
func (s *JSONStorage) load() error {
	s.data = &PostedData{Posted: make(map[string]PostedEntry)}

	bytes, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			// ファイルが存在しない場合は初期状態として許容
			return nil
		}
		return err
	}

	if len(bytes) == 0 {
		// ファイルが空の場合も初期状態として許容
		return nil
	}

	return json.Unmarshal(bytes, s.data)
}

// IsPosted はIDが投稿済みかチェックする
func (s *JSONStorage) IsPosted(paperID string) bool {
	_, exists := s.data.Posted[paperID]
	return exists
}

// Add は新しい投稿記録を追加する
func (s *JSONStorage) Add(paperID, venue string) {
	s.data.Posted[paperID] = PostedEntry{
		Date:  time.Now().UTC().Format("2006-01-02"),
		Venue: venue,
	}
}

// Save は現在のデータをJSONファイルに書き込む
func (s *JSONStorage) Save() error {
	// ディレクトリが存在しない場合に作成
	dir := filepath.Dir(s.path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	bytes, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, bytes, 0644)
}
