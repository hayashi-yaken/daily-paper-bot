package selector

import (
	"errors"
	"math/rand"
	"testing"
)

// MockPaper はテスト用のPaper実装です。
type MockPaper struct {
	id    string
	title string
}

func (m *MockPaper) GetID() string {
	return m.id
}

func (m *MockPaper) GetTitle() string {
	return m.title
}

func TestRandomSelector_Select(t *testing.T) {
	papers := []Paper{
		&MockPaper{id: "p1", title: "Title 1"},
		&MockPaper{id: "p2", title: "Title 2"},
		&MockPaper{id: "p3", title: "Title 3"},
		&MockPaper{id: "p4", title: ""},      // Invalid title
		&MockPaper{id: "", title: "Title 5"}, // Invalid ID
	}

	t.Run("select one from valid candidates", func(t *testing.T) {
		selector := NewRandomSelector()
		// 乱数を固定
		selector.rand = rand.New(rand.NewSource(0))

		selected, err := selector.Select(papers)
		if err != nil {
			t.Fatalf("Select() returned an error: %v", err)
		}

		// 有効候補は p1, p2, p3 の 3 件。seed 0 では Intn(3)=0 なので p1 が選ばれる
		expectedID := "p1"
		if selected.GetID() != expectedID {
			t.Errorf("expected paper %s, but got %s", expectedID, selected.GetID())
		}
	})

	t.Run("no candidates because of invalid data", func(t *testing.T) {
		papersOnlyInvalid := []Paper{
			&MockPaper{id: "", title: "Title 1"},
			&MockPaper{id: "p2", title: ""},
		}
		selector := NewRandomSelector()

		_, err := selector.Select(papersOnlyInvalid)
		if !errors.Is(err, ErrNoCandidates) {
			t.Errorf("expected ErrNoCandidates, but got %v", err)
		}
	})

	t.Run("no papers provided", func(t *testing.T) {
		selector := NewRandomSelector()
		_, err := selector.Select([]Paper{})
		if !errors.Is(err, ErrNoCandidates) {
			t.Errorf("expected ErrNoCandidates, but got %v", err)
		}
	})
}
