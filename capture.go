package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

var (
	seenHashes   = map[string]bool{}
	currentGroup string
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
