package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/getlantern/systray"
)

var (
	mStatus  *systray.MenuItem
	mLastMsg *systray.MenuItem

	ringBuffer = newTextRing(5 * 60)
)

func onReady() {
	systray.SetTemplateIcon(iconData, iconData)
	systray.SetTitle("qqcliput")
	systray.SetTooltip("qqcliput — QQ 消息监控")

	mStatus = systray.AddMenuItem("状态: 启动中...", "")
	mStatus.Disable()
	mLastMsg = systray.AddMenuItem("暂无消息", "")
	mLastMsg.Disable()
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("退出", "退出程序")

	ctx, cancel := context.WithCancel(context.Background())

	go captureLoop(ctx)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		cancel()
		systray.Quit()
	}()

	go func() {
		<-mQuit.ClickedCh
		cancel()
		systray.Quit()
	}()
}

func onExit() {
	fmt.Println("qqcliput 已退出")
}

func main() {
	systray.Run(onReady, onExit)
}
