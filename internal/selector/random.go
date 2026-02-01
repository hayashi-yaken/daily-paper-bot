package selector

import (
	"errors"
	"math/rand"
	"time"
)

var ErrNoCandidates = errors.New("no candidates to select from")

// RandomSelector はランダムに論文を選定するセレクターです。
type RandomSelector struct {
	// isPosted は論文が投稿済みか判定する関数です。
	isPosted func(paperID string) bool
	// rand はテストで乱数生成を固定できるようにインターフェースとして保持します。
	rand *rand.Rand
}

// NewRandomSelector は新しいRandomSelectorを生成します。
func NewRandomSelector(isPosted func(paperID string) bool) *RandomSelector {
	return &RandomSelector{
		isPosted: isPosted,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Select は論文リストから未投稿のものをフィルタリングし、ランダムに1本を選定します。
func (s *RandomSelector) Select(papers []Paper) (Paper, error) {
	var candidates []Paper

	// 投稿済みを除外し、必須項目をチェックします。
	for _, p := range papers {
		if p.GetID() == "" || p.GetTitle() == "" {
			continue // データ不整合はスキップ
		}
		if !s.isPosted(p.GetID()) {
			candidates = append(candidates, p)
		}
	}

	if len(candidates) == 0 {
		return nil, ErrNoCandidates
	}

	// ランダムに1本選定します。
	selectedIndex := s.rand.Intn(len(candidates))

	return candidates[selectedIndex], nil
}
