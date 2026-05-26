package selector

import (
	"errors"
	"math/rand"
	"time"
)

var ErrNoCandidates = errors.New("no candidates to select from")

// RandomSelector はランダムに論文を選定するセレクターです。
type RandomSelector struct {
	// rand はテストで乱数生成を固定できるようにインターフェースとして保持します。
	rand *rand.Rand
}

// NewRandomSelector は新しいRandomSelectorを生成します。
func NewRandomSelector() *RandomSelector {
	return &RandomSelector{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Select は論文リストから必須項目が揃ったものをフィルタリングし、ランダムに1本を選定します。
func (s *RandomSelector) Select(papers []Paper) (Paper, error) {
	var candidates []Paper

	// 必須項目をチェックします。
	for _, p := range papers {
		if p == nil {
			continue
		}
		if p.GetID() == "" || p.GetTitle() == "" {
			continue // データ不整合はスキップ
		}
		candidates = append(candidates, p)
	}

	if len(candidates) == 0 {
		return nil, ErrNoCandidates
	}

	// ランダムに1本選定します。
	selectedIndex := s.rand.Intn(len(candidates))

	return candidates[selectedIndex], nil
}
