package main

import (
	"context"
	"fmt"
	"time"
)

var lastText string

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

		text := cOCRWindow(wid)
		if text == "" {
			continue
		}

		if text != lastText {
			now := time.Now()
			fmt.Printf("[%s] %s\n", now.Format("15:04:05"), text)
			lastText = text
			mLastMsg.SetTitle(fmt.Sprintf("消息: %s", now.Format("15:04:05")))
			ringBuffer.Append(now, text)
		}
	}
}
