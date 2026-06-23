package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
)

type Block struct {
	Text string  `json:"text"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	W    float64 `json:"w"`
	H    float64 `json:"h"`
}

type Message struct {
	Group    string `json:"group,omitempty"`
	Sender   string `json:"sender,omitempty"`
	Time     string `json:"time,omitempty"`
	Type     string `json:"type"`
	Content  string `json:"content,omitempty"`
	Duration string `json:"duration,omitempty"`
}

var (
	reTimeAMPM = regexp.MustCompile(`(上午|下午|早上|晚上)\s*\d{1,2}:\d{2}`)
	reTimeHM   = regexp.MustCompile(`\d{1,2}:\d{2}`)
	reDuration = regexp.MustCompile(`\d{1,3}:\d{2}`)
	reDigits   = regexp.MustCompile(`^\d{1,2}$`)
	reColon    = regexp.MustCompile(`^:\d{1,2}$`)
)

func parseOCRJSON(raw string) []Block {
	var blocks []Block
	if err := json.Unmarshal([]byte(raw), &blocks); err != nil {
		return nil
	}
	return blocks
}

func mergeTimeFragments(blocks []Block) []Block {
	if len(blocks) < 2 {
		return blocks
	}

	sorted := make([]Block, len(blocks))
	copy(sorted, blocks)
	sort.Slice(sorted, func(i, j int) bool {
		if math.Abs(sorted[i].Y-sorted[j].Y) > 0.015 {
			return sorted[i].Y > sorted[j].Y
		}
		return sorted[i].X < sorted[j].X
	})

	merged := make([]Block, 0, len(sorted))
	skip := false
	for i := 0; i < len(sorted); i++ {
		if skip {
			skip = false
			continue
		}

		t := strings.TrimSpace(sorted[i].Text)
		if reDigits.MatchString(t) && i+1 < len(sorted) {
			nextT := strings.TrimSpace(sorted[i+1].Text)
			if reColon.MatchString(nextT) {
				sorted[i].Text = t + nextT
				merged = append(merged, sorted[i])
				skip = true
				continue
			}
		}

		merged = append(merged, sorted[i])
	}
	return merged
}

func autoChatRange(blocks []Block) (minX, maxX float64) {
	if len(blocks) < 2 {
		return 0, 1
	}

	minX = 1.0
	maxX = 0.0
	found := false

	for _, b := range blocks {
		if b.X < 0.5 && b.X+b.W > 0.5 {
			if b.X < minX {
				minX = b.X
			}
			if b.X+b.W > maxX {
				maxX = b.X + b.W
			}
			found = true
		}
	}

	if !found {
		bestDist := 1.0
		for _, b := range blocks {
			cx := b.X + b.W/2
			dist := math.Abs(cx - 0.5)
			if dist < bestDist {
				bestDist = dist
				minX = b.X
				maxX = b.X + b.W
			}
		}
	}

	minX -= 0.04
	maxX += 0.04
	if minX < 0 {
		minX = 0
	}
	if maxX > 1 {
		maxX = 1
	}

	return
}

func filterChatArea(blocks []Block) []Block {
	minX, maxX := autoChatRange(blocks)
	var out []Block
	for _, b := range blocks {
		cx := b.X + b.W/2
		if cx > minX && cx < maxX {
			out = append(out, b)
		}
	}
	return out
}

func clusterBlocks(blocks []Block) [][]Block {
	if len(blocks) == 0 {
		return nil
	}

	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].Y > blocks[j].Y
	})

	var groups [][]Block
	cur := []Block{blocks[0]}
	curAvgCenter := blocks[0].Y - blocks[0].H/2
	curCount := 1

	for i := 1; i < len(blocks); i++ {
		b := blocks[i]
		bCenter := b.Y - b.H/2

		if math.Abs(curAvgCenter-bCenter) < 0.035 {
			cur = append(cur, b)
			curAvgCenter = (curAvgCenter*float64(curCount) + bCenter) / float64(curCount+1)
			curCount++
		} else {
			groups = append(groups, cur)
			cur = []Block{b}
			curAvgCenter = bCenter
			curCount = 1
		}
	}
	groups = append(groups, cur)

	return groups
}

func splitSenderContent(text string) (sender, content string, ok bool) {
	idx := strings.Index(text, "：")
	if idx > 0 && idx <= 16 {
		return strings.TrimSpace(text[:idx]), strings.TrimSpace(text[idx+3:]), true
	}
	idx = strings.Index(text, ":")
	if idx > 0 && idx <= 16 {
		return strings.TrimSpace(text[:idx]), strings.TrimSpace(text[idx+1:]), true
	}
	return "", "", false
}

func inferGroupTitle(group []Block) (string, bool) {
	if len(group) != 1 {
		return "", false
	}
	b := group[0]
	t := strings.TrimSpace(b.Text)
	if t == "" {
		return "", false
	}
	if isTimeString(t) {
		return "", false
	}
	if strings.HasPrefix(t, "[") {
		return "", false
	}
	cx := b.X + b.W/2
	if cx < 0.30 || cx > 0.70 {
		return "", false
	}
	if len([]rune(t)) < 4 || len([]rune(t)) > 25 {
		return "", false
	}
	return t, true
}

func buildMessage(blocks []Block) Message {
	if len(blocks) == 0 {
		return Message{Type: "empty"}
	}

	sorted := make([]Block, len(blocks))
	copy(sorted, blocks)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].X < sorted[j].X
	})

	var sender, timeStr, content string
	senderFound := false

	for i, b := range sorted {
		t := strings.TrimSpace(b.Text)
		if t == "" {
			continue
		}
		if isTimeString(t) {
			timeStr = t
			continue
		}
		if !senderFound && isSenderString(t, b.X) {
			if len([]rune(t)) <= 3 && i+1 < len(sorted) {
				if sorted[i+1].X-(b.X+b.W) < 0.03 {
					if content != "" {
						content += " "
					}
					content += t
					continue
				}
			}
			sender = t
			senderFound = true
			continue
		}
		if content != "" {
			content += " "
		}
		content += t
	}

	content = strings.TrimSpace(content)
	msgType := detectType(sender, timeStr, content, blocks)
	duration := ""
	if msgType == "video" {
		duration = extractDuration(content, blocks)
	}

	return Message{
		Sender:   sender,
		Time:     timeStr,
		Type:     msgType,
		Content:  content,
		Duration: duration,
	}
}

func inferMessages(group []Block) []Message {
	if len(group) == 0 {
		return nil
	}

	// Single block
	if len(group) == 1 {
		t := strings.TrimSpace(group[0].Text)
		if t == "" {
			return nil
		}
		if isTimeString(t) {
			return []Message{{Time: t, Type: "text"}}
		}
		sender, content, ok := splitSenderContent(t)
		if !ok {
			content = t
		}
		msgType := detectType(sender, "", content, group)
		duration := ""
		if msgType == "video" {
			duration = extractDuration(content, group)
		}
		return []Message{{Sender: sender, Time: "", Type: msgType, Content: content, Duration: duration}}
	}

	// Multiple blocks: detect if cluster contains multiple messages
	xSorted := make([]Block, len(group))
	copy(xSorted, group)
	sort.Slice(xSorted, func(i, j int) bool {
		return xSorted[i].X < xSorted[j].X
	})

	type cand struct{ yAvg float64 }
	var cands []cand
	for _, b := range xSorted {
		t := strings.TrimSpace(b.Text)
		if t == "" {
			continue
		}
		if isTimeString(t) {
			continue
		}
		if strings.HasPrefix(t, "[") {
			continue
		}
		if len([]rune(t)) > 10 {
			continue
		}
		if b.X > 0.35 {
			continue
		}
		if len(t) <= 4 {
			allDigit := true
			for _, r := range t {
				if r < '0' || r > '9' {
					allDigit = false
					break
				}
			}
			if allDigit {
				continue
			}
		}
		cands = append(cands, cand{b.Y - b.H/2})
	}

	if len(cands) >= 2 {
		sort.Slice(cands, func(p, q int) bool {
			return cands[p].yAvg > cands[q].yAvg
		})
		maxGap := 0.0
		splitY := 0.0
		for p := 0; p < len(cands)-1; p++ {
			gap := cands[p].yAvg - cands[p+1].yAvg
			if gap > maxGap {
				maxGap = gap
				splitY = (cands[p].yAvg + cands[p+1].yAvg) / 2
			}
		}

		if splitY > 0 {
			var sub1, sub2 []Block
			for _, b := range group {
				if b.Y >= splitY {
					sub1 = append(sub1, b)
				} else {
					sub2 = append(sub2, b)
				}
			}
			if len(sub1) > 0 && len(sub2) > 0 {
				var msgs []Message
				msgs = append(msgs, inferMessages(sub1)...)
				msgs = append(msgs, inferMessages(sub2)...)
				return msgs
			}
		}
	}

	return []Message{buildMessage(group)}
}

func isTimeString(s string) bool {
	if reTimeAMPM.MatchString(s) {
		return true
	}
	if reTimeHM.MatchString(s) && len(s) <= 6 {
		return true
	}
	return false
}

func isSenderString(s string, x float64) bool {
	if len([]rune(s)) > 10 {
		return false
	}
	if x > 0.45 {
		return false
	}
	if isTimeString(s) {
		return false
	}
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "[") {
		return false
	}
	// Reject pure numbers (OCR fragments from time stamps)
	if len(s) <= 4 {
		allDigits := true
		for _, r := range s {
			if r < '0' || r > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return false
		}
	}
	return true
}

func detectType(sender, timeStr, content string, group []Block) string {
	allText := sender + " " + timeStr + " " + content
	maxY := 0.0
	for _, b := range group {
		if b.Y > maxY {
			maxY = b.Y
		}
	}

	// 系统消息通常居中（X 在 0.2-0.8），且单行
	if len(group) <= 1 && maxY > 0.7 && content != "" {
		centerX := group[0].X + group[0].W/2
		if centerX > 0.25 && centerX < 0.75 && len([]rune(content)) > 4 {
			return "system"
		}
	}

	switch {
	case strings.Contains(allText, "[视频]") || reDuration.MatchString(allText):
		return "video"
	case strings.Contains(allText, "[图片]"):
		return "image"
	case strings.Contains(allText, "[文件]") || strings.Contains(allText, "[文件]"):
		return "file"
	case strings.Contains(allText, "[表情]") || strings.Contains(allText, "[Sticker]") || strings.Contains(allText, "[动画]"):
		return "sticker"
	case strings.Contains(allText, "[红包]"):
		return "redpacket"
	case content == "" && sender != "":
		return "image"
	case content == "":
		return "unknown"
	default:
		return "text"
	}
}

func extractDuration(content string, group []Block) string {
	matches := reDuration.FindString(content)
	if matches != "" {
		return matches
	}
	for _, b := range group {
		matches := reDuration.FindString(b.Text)
		if matches != "" {
			return matches
		}
	}
	return ""
}

func blockHash(m Message) string {
	data := fmt.Sprintf("%s|%s|%s|%s", m.Sender, m.Time, m.Type, m.Content)
	h := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", h[:8])
}
