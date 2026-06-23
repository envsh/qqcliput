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

const yThreshold = 0.025

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

func clusterBlocks(blocks []Block) [][]Block {
	if len(blocks) == 0 {
		return nil
	}

	sort.Slice(blocks, func(i, j int) bool {
		if math.Abs(blocks[i].Y-blocks[j].Y) < yThreshold/2 {
			return blocks[i].X < blocks[j].X
		}
		return blocks[i].Y > blocks[j].Y
	})

	var groups [][]Block
	cur := []Block{blocks[0]}

	for i := 1; i < len(blocks); i++ {
		lastY := cur[len(cur)-1].Y
		if math.Abs(blocks[i].Y-lastY) < yThreshold {
			cur = append(cur, blocks[i])
		} else {
			groups = append(groups, cur)
			cur = []Block{blocks[i]}
		}
	}
	groups = append(groups, cur)

	return groups
}

func inferGroupTitle(group []Block) (string, bool) {
	if len(group) != 1 {
		return "", false
	}
	b := group[0]
	if b.X > 0.3 && b.X < 0.8 && b.Text != "" {
		return strings.TrimSpace(b.Text), true
	}
	return "", false
}

func inferMessage(group []Block) Message {
	if len(group) == 0 {
		return Message{Type: "empty"}
	}

	var sender, timeStr, content string

	for _, b := range group {
		t := strings.TrimSpace(b.Text)
		if t == "" {
			continue
		}

		if isTimeString(t) {
			timeStr = t
		} else if isSenderString(t, b.X) {
			sender = t
		} else {
			if content != "" {
				content += " "
			}
			content += t
		}
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
	if len([]rune(s)) > 8 {
		return false
	}
	if x > 0.35 {
		return false
	}
	if isTimeString(s) {
		return false
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
