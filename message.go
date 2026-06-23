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

const (
	clusterOverlap = 0.001
)

var (
	reTimeAMPM = regexp.MustCompile(`(上午|下午|早上|晚上)\d{1,2}:\d{2}`)
	reTimeHM   = regexp.MustCompile(`\d{1,2}:\d{2}`)
	reDuration = regexp.MustCompile(`\d{1,3}:\d{2}`)
)

func parseOCRJSON(raw string) []Block {
	var blocks []Block
	if err := json.Unmarshal([]byte(raw), &blocks); err != nil {
		return nil
	}
	return blocks
}

func autoChatRange(blocks []Block) (minX, maxX float64) {
	if len(blocks) < 2 {
		return 0, 1
	}

	// Chat messages are WIDE blocks that span across the center (X=0.5).
	// Sidebar and right panel blocks are narrow and don't cross X=0.5.
	// Find blocks that span X=0.5 to locate the chat area boundaries.
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
		// Fallback: use the block whose center is closest to X=0.5
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

	// Add padding to include sender names (left) and timestamps (right)
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
	curTop := blocks[0].Y - blocks[0].H
	curBot := blocks[0].Y

	for i := 1; i < len(blocks); i++ {
		b := blocks[i]
		bTop := b.Y - b.H
		bBot := b.Y

		if bTop < curBot-clusterOverlap && bBot > curTop+clusterOverlap {
			cur = append(cur, b)
			if bTop < curTop {
				curTop = bTop
			}
			if bBot > curBot {
				curBot = bBot
			}
		} else {
			groups = append(groups, cur)
			cur = []Block{b}
			curTop, curBot = bTop, bBot
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

func inferMessage(group []Block) Message {
	if len(group) == 0 {
		return Message{Type: "empty"}
	}

	// Single block: try splitting "Sender：Content"
	if len(group) == 1 {
		t := strings.TrimSpace(group[0].Text)
		if t == "" {
			return Message{Type: "empty"}
		}
		if isTimeString(t) {
			return Message{Time: t, Type: "text"}
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
		return Message{Sender: sender, Time: "", Type: msgType, Content: content, Duration: duration}
	}

	// Multiple blocks: leftmost short block is sender, rest is content
	sorted := make([]Block, len(group))
	copy(sorted, group)
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
			// Very short text that's close to next block: OCR fragment, not sender
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
	msgType := detectType(sender, timeStr, content, group)
	duration := ""
	if msgType == "video" {
		duration = extractDuration(content, group)
	}

	return Message{
		Sender:   sender,
		Time:     timeStr,
		Type:     msgType,
		Content:  content,
		Duration: duration,
	}
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
