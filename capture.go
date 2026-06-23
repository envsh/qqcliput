package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

var (
	seenHashes   = map[string]bool{}
	currentGroup string
	firstOCR     = true
)

func setStatus(text string) {
	mStatus.SetTitle("状态: " + text)
}

func captureLoop(ctx context.Context) {
	setStatus("等待 QQ 窗口...")

	widCh := make(chan uint32, 1)
	go func() {
		widCh <- cFindQQWindow()
	}()

	var wid uint32
	select {
	case wid = <-widCh:
	case <-ctx.Done():
		return
	}

	fmt.Fprintf(os.Stderr, "=== qqcliput started ===\n")
	w, h := cGetWindowBounds(wid)
	fmt.Fprintf(os.Stderr, "QQ window detected: id=%d, size=%dx%d\n", wid, w, h)

	setStatus("运行中")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		if !cIsQQFrontmost() {
			if !cWindowExists(wid) {
				setStatus("等待 QQ 窗口...")
				go func() {
					widCh <- cFindQQWindow()
				}()
				select {
				case wid = <-widCh:
					fmt.Fprintf(os.Stderr, "=== qqcliput started ===\n")
					w, h := cGetWindowBounds(wid)
					fmt.Fprintf(os.Stderr, "QQ window detected: id=%d, size=%dx%d\n", wid, w, h)

					setStatus("运行中")
				case <-ctx.Done():
					return
				}
			}
			continue
		}

		raw := cOCRWindowJSON(wid)
		if raw == "[]" || raw == "" {
			continue
		}

		blocks := parseOCRJSON(raw)
		blocks = filterChatArea(blocks)
		if firstOCR && len(blocks) > 0 {
			rawBlocks := parseOCRJSON(raw)
			minX, maxX := autoChatRange(rawBlocks)
			fmt.Fprintf(os.Stderr, "=== Auto-detected chat area: X %.2f ~ %.2f ===\n", minX, maxX)
			const nb = 50
			var buckets [nb]int
			for _, b := range rawBlocks {
				cx := b.X + b.W/2
				idx := int(cx / 0.02)
				if idx >= nb {
					idx = nb - 1
				}
				buckets[idx]++
			}
			fmt.Fprintf(os.Stderr, "=== Density:")
			for i, v := range buckets {
				if v > 0 {
					fmt.Fprintf(os.Stderr, " %.2f:%d", float64(i)*0.02, v)
				}
			}
			fmt.Fprintf(os.Stderr, " ===\n")
			firstOCR = false
		}
		if len(blocks) == 0 {
			continue
		}

		groups := clusterBlocks(blocks)
		now := time.Now()
		anyNew := false

		if len(groups) > 0 {
			name, ok := inferGroupTitle(groups[0])
			if ok {
				currentGroup = name
			}
		}

		for i := len(groups) - 1; i >= 0; i-- {
			msg := inferMessage(groups[i])
			if msg.Type == "empty" {
				continue
			}

			if name, ok := inferGroupTitle(groups[i]); ok {
				currentGroup = name
				continue
			}

			if msg.Sender == "" && msg.Content == "" {
				continue
			}

			msg.Group = currentGroup
			hash := blockHash(msg)
			if seenHashes[hash] {
				continue
			}
			seenHashes[hash] = true

			b, err := json.Marshal(msg)
			if err != nil {
				continue
			}
			fmt.Println(string(b))
			ringBuffer.Append(now, string(b))
			anyNew = true
		}

		if anyNew {
			mLastMsg.SetTitle(fmt.Sprintf("消息: %s", now.Format("15:04:05")))
		}
	}
}
